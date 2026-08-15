-- Promotions and the hive club (docs/PLAN_ERA_2.md, phase E7).
--
-- What is deliberately NOT here: a membership table. The design's own
-- sign-in screen defines the Hive club as HAVING AN ACCOUNT ("Create an
-- account — first order ships free", "8% less on every order after the
-- first"), so both member perks derive from a count of the user's
-- non-cancelled orders — a fact the orders table already holds. A
-- users.tier column would be a denormalized copy of that count's sign, and
-- E5's variant_prices migration showed what synced copies do: drift.
-- Decision #66 in docs/ARCHITECTURE.md.

-- ── Promo codes ───────────────────────────────────────────────────────────
--
-- The code's IDENTITY and market-invariant rules. Everything that is money
-- lives in promo_code_values below, per market — a fixed discount and a
-- minimum-subtotal floor are shelf prices in the E5 sense, and one number
-- converted at a rate would move with the rate.
CREATE TABLE promo_codes (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT   NOT NULL,

    kind TEXT NOT NULL CHECK (kind IN ('percent', 'fixed', 'free_shipping')),
    -- Present exactly when kind = 'percent'. The `=` between two booleans is
    -- a biconditional: a percent code must carry a percent, every other kind
    -- must not. One CHECK states both directions.
    percent SMALLINT CHECK (percent BETWEEN 1 AND 100),
    CHECK ((kind = 'percent') = (percent IS NOT NULL)),

    -- NULL bound = unbounded on that side. Half-open windows are real:
    -- "valid from Black Friday" and "valid until the jars run out" both
    -- exist, and neither should have to invent a fake far-future date.
    starts_at TIMESTAMPTZ,
    ends_at   TIMESTAMPTZ,
    CHECK (starts_at IS NULL OR ends_at IS NULL OR starts_at < ends_at),

    -- NULL = uncapped. The cap is enforced under a row lock in CreateOrder,
    -- not here — a CHECK cannot count rows in another table.
    max_redemptions INT CHECK (max_redemptions > 0),

    active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Customers type codes, and "honey10" must find HONEY10: uniqueness is on
-- the UPPERCASED expression, so two codes differing only in case cannot
-- coexist and every lookup normalizes the same way. An expression index is
-- the same idea as an IMMUTABLE computed key — upper() of a stored value.
CREATE UNIQUE INDEX idx_promo_codes_code ON promo_codes (upper(code));

-- ── Per-market money for a code ───────────────────────────────────────────
--
-- The variant_prices shape a third time (variants, shipping rates, now
-- promos): money keyed (owner, currency), no conversion fallback. A fixed
-- code with no amount row for the shopper's market cannot be applied there
-- — refused honestly, not converted quietly (the same degrade/refuse rule
-- as everything E5 touched).
CREATE TABLE promo_code_values (
    code_id  BIGINT NOT NULL REFERENCES promo_codes (id) ON DELETE CASCADE,
    currency TEXT   NOT NULL REFERENCES currencies (code),

    -- The discount, for kind = 'fixed'.
    amount_minor BIGINT CHECK (amount_minor > 0),
    -- "Valid on orders over X" — meaningful for every kind.
    min_subtotal_minor BIGINT CHECK (min_subtotal_minor > 0),

    PRIMARY KEY (code_id, currency)
);

-- ── Redemptions ───────────────────────────────────────────────────────────
--
-- One row per code actually used on an order. The UNIQUE (code_id, user_id)
-- index is the phase's concurrency lesson in one line: "once per customer"
-- is not code that checks a count — it is a property of the storage, and
-- two racing checkouts that both try to redeem hit the index, not a window
-- between a SELECT and an INSERT. The GLOBAL cap (max_redemptions) cannot
-- be an index (it is a count, not a key), so it is enforced like stock:
-- under a FOR UPDATE lock on the promo row.
--
-- ON DELETE RESTRICT for the order: order rows are never deleted in this
-- schema, and a redemption silently outliving its order would break the
-- release-on-cancel rule below. Cancelling an order DELETEs its redemption
-- row explicitly (UpdateOrderStatus), returning the code to the customer
-- the same way cancelling returns stock to the shelf.
CREATE TABLE promo_redemptions (
    id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code_id  BIGINT NOT NULL REFERENCES promo_codes (id) ON DELETE CASCADE,
    user_id  BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    order_id BIGINT NOT NULL UNIQUE REFERENCES orders (id) ON DELETE RESTRICT,

    redeemed_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (code_id, user_id)
);

-- ── The promo a cart is carrying ──────────────────────────────────────────
--
-- Applying a code is server-side cart state, not a string the client resends
-- with every request: the customer applies it on the cart page and it must
-- still be there on the checkout page, in another tab, after a reload. One
-- row per user (the PK), because a cart carries at most one code — the
-- design draws one promo box, not a list.
--
-- Validity is deliberately NOT checked by this table. A code can expire or
-- sell out BETWEEN apply and checkout, so validity is evaluated on every
-- read (preview) and re-checked under lock at checkout — this row only
-- remembers the choice.
CREATE TABLE cart_promos (
    user_id    BIGINT      PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    code_id    BIGINT      NOT NULL REFERENCES promo_codes (id) ON DELETE CASCADE,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ── The order's discount, split ───────────────────────────────────────────
--
-- E6's discount_minor stays the one figure in the balance CHECK; these two
-- are its composition, because the receipt draws them as separate lines
-- ("Hive club discount" / the promo by name) and a single number could not
-- say which promise produced it. DEFAULT 0 backfills every existing order
-- truthfully: their discount was zero, so 0 + 0 = 0 holds.
--
-- promo_code is a SNAPSHOT of the code's text, same rule as product names
-- in order_items: the receipt must keep saying "HONEY10" even if the family
-- later renames or deletes the code. NULL = no promo on this order.
ALTER TABLE orders
    ADD COLUMN member_discount_minor BIGINT NOT NULL DEFAULT 0 CHECK (member_discount_minor >= 0),
    ADD COLUMN promo_discount_minor  BIGINT NOT NULL DEFAULT 0 CHECK (promo_discount_minor >= 0),
    ADD COLUMN promo_code            TEXT;

ALTER TABLE orders
    ADD CONSTRAINT orders_discount_split
        CHECK (member_discount_minor + promo_discount_minor = discount_minor);
