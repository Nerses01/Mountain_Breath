-- Real checkout: address book, order snapshots, the money breakdown and
-- shipping rates (docs/PLAN_ERA_2.md, phase E6).

-- ── The address book ──────────────────────────────────────────────────────
--
-- The customer's EDITABLE addresses. Editable is the operative word, and it
-- is why the orders table below does NOT reference this one: an order points
-- at nothing it does not own. E6 keeps the book to one default row per user
-- (the checkout form upserts it, so the next checkout is pre-filled); E8's
-- account page is where "several named addresses" becomes worth building.
CREATE TABLE addresses (
    id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    first_name  TEXT NOT NULL,
    last_name   TEXT NOT NULL,
    phone       TEXT NOT NULL,
    street      TEXT NOT NULL,
    city        TEXT NOT NULL,
    postal_code TEXT NOT NULL,
    country     TEXT NOT NULL,

    is_default BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One default per user — the same constant-expression partial unique index
-- as currencies' one-base rule (000016), partitioned per user like the
-- one-primary-image rule (000011). Third use of the pattern; it is now an
-- idiom of this schema.
CREATE UNIQUE INDEX idx_addresses_one_default
    ON addresses (user_id)
    WHERE is_default;

-- ── The order's own copy of everything ────────────────────────────────────
--
-- Snapshot columns, NOT a foreign key to addresses: decision #3 (prices),
-- E5 (currency), and now the address — an order is a closed record of a
-- transaction, and nothing the customer can later edit may reach into it.
-- Flat columns rather than JSONB for the same reason products got columns:
-- constraints and honest NULLs beat a blob that can hold anything.
--
-- Nullable, because orders created before E6 genuinely had no address —
-- NULL is the true statement about them. The API layer requires an address
-- for every NEW order.
ALTER TABLE orders
    ADD COLUMN ship_first_name  TEXT,
    ADD COLUMN ship_last_name   TEXT,
    ADD COLUMN ship_phone       TEXT,
    ADD COLUMN ship_street      TEXT,
    ADD COLUMN ship_city        TEXT,
    ADD COLUMN ship_postal_code TEXT,
    ADD COLUMN ship_country     TEXT,
    ADD COLUMN delivery_note    TEXT NOT NULL DEFAULT '',
    ADD COLUMN leave_with_neighbour BOOLEAN NOT NULL DEFAULT FALSE;

-- ── The money breakdown ───────────────────────────────────────────────────
--
-- One total becomes five figures, all in the order's currency (E5), all in
-- minor units. Added with DEFAULT 0 so the ~backfill~ is the default itself:
-- an old order's total WAS its subtotal, shipping and discount were zero.
ALTER TABLE orders
    ADD COLUMN subtotal_minor BIGINT NOT NULL DEFAULT 0 CHECK (subtotal_minor >= 0),
    ADD COLUMN shipping_minor BIGINT NOT NULL DEFAULT 0 CHECK (shipping_minor >= 0),
    ADD COLUMN discount_minor BIGINT NOT NULL DEFAULT 0 CHECK (discount_minor >= 0),
    ADD COLUMN tax_minor      BIGINT NOT NULL DEFAULT 0 CHECK (tax_minor >= 0);

UPDATE orders SET subtotal_minor = total_minor;

-- The arithmetic, enforced where it cannot be forgotten. A CHECK constraint
-- is the database's assert(): every INSERT and UPDATE proves the books
-- balance, including ones made by a future bug, a migration, or somebody in
-- psql at midnight.
--
-- tax_minor is deliberately NOT in the sum. "Prices include VAT" (§1.4,
-- Armenian retail convention): the tax is CONTAINED in the subtotal and
-- only broken out for the invoice, so adding it to the total would charge
-- it twice. The second CHECK pins that meaning — a contained tax can never
-- exceed what contains it.
ALTER TABLE orders
    ADD CONSTRAINT orders_totals_balance
        CHECK (subtotal_minor + shipping_minor - discount_minor = total_minor),
    ADD CONSTRAINT orders_tax_contained
        CHECK (tax_minor <= subtotal_minor);

-- ── Payment: a method and a status, two separate facts ────────────────────
--
-- The method is HOW the customer chose to pay; the status is WHETHER money
-- has arrived. Folding them into the order-status state machine would be
-- wrong twice over: a bank transfer is 'confirmed' long before it is 'paid',
-- and a cash-on-delivery order is 'delivered' at the exact moment it stops
-- being 'unpaid'. Orthogonal states get orthogonal columns.
--
-- Card is a STUB — it records the intention and lands unpaid, like the
-- others; the real provider (Idram / Ameriabank vPOS) is Phase 11 work.
ALTER TABLE orders
    ADD COLUMN payment_method TEXT NOT NULL DEFAULT 'card'
        CHECK (payment_method IN ('card', 'bank_transfer', 'cash_on_delivery')),
    ADD COLUMN payment_status TEXT NOT NULL DEFAULT 'unpaid'
        CHECK (payment_status IN ('unpaid', 'paid', 'refunded'));

-- The method default existed only to backfill pre-E6 rows ('card' is as
-- honest as any guess about orders that never chose). Dropping it makes a
-- future INSERT that forgets the method a NOT NULL violation instead of a
-- silent guess — the same move as 000016's currency default.
ALTER TABLE orders ALTER COLUMN payment_method DROP DEFAULT;

-- ── Shipping rates ────────────────────────────────────────────────────────
--
-- A table, not constants in code, because the family will change these
-- without a deploy. Keyed by (method, currency): a rate is money, and E5's
-- lesson applies to it unchanged — a shipping fee is a per-market shelf
-- price, set by hand, not one number converted. There is deliberately no
-- conversion fallback here: a market the shop ships to gets a real rate row,
-- and checkout refuses a market that has none (same degrade/refuse split as
-- variant prices — browsing a market is free, charging it is not).
--
--   base_minor                what standard delivery costs
--   cold_chain_surcharge_minor  added when ANY item in the order is
--                             cold-chain (products.is_cold_chain, E3) —
--                             the design's "Chilled shipping" line
--   free_over_minor           subtotal at which the BASE becomes free; the
--                             surcharge survives, because the chilled box
--                             costs the family real money either way. That
--                             reading is §1.4's cart arithmetic taken
--                             literally: the mock charges $6 chilled
--                             shipping on a subtotal past its threshold.
CREATE TABLE shipping_rates (
    method   TEXT NOT NULL,
    currency TEXT NOT NULL REFERENCES currencies (code),

    base_minor                 BIGINT NOT NULL CHECK (base_minor >= 0),
    cold_chain_surcharge_minor BIGINT NOT NULL DEFAULT 0 CHECK (cold_chain_surcharge_minor >= 0),
    -- NULL = no free-shipping threshold in this market.
    free_over_minor            BIGINT CHECK (free_over_minor > 0),

    PRIMARY KEY (method, currency)
);

-- Bootstrap rows, same justification as 000016's fx rate: without them no
-- checkout can be quoted, so the schema would ship broken-by-default. The
-- figures are placeholder like every price in this project (§1.1); the
-- family edits the table, which is the point of it being a table.
--
-- $4 / 1,900 ֏ standard; $6 / 2,900 ֏ chilled surcharge; free base over
-- $70 / 33,500 ֏ (the threshold §1.4 derived from "$8 away" on a $62 cart).
INSERT INTO shipping_rates (method, currency, base_minor, cold_chain_surcharge_minor, free_over_minor)
VALUES ('standard', 'USD',  400,  600,  7000),
       ('standard', 'AMD', 1900, 2900, 33500);
