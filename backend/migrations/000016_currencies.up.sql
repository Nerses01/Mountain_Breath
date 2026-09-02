-- Dual currency: USD and AMD (docs/PLAN_ERA_2.md, phase E5).
--
-- The idea this migration is built on: a price is not a property of a
-- product. It is a property of a (product, market) PAIR. Everything below
-- follows from taking that literally.

-- ── The markets the shop serves ───────────────────────────────────────────
--
-- A table rather than a Go enum because the shop's SQL has to ROUND with
-- these numbers (see the view at the bottom), and a rounding rule that lives
-- only in application code is a rounding rule the database can silently
-- disagree with.
--
-- minor_exponent is the reason `price_minor / 100` is a bug and not a
-- convention. USD has 2 (100 cents to the dollar); AMD has 0 — a dram has no
-- subdivision in circulation, so its "minor unit" IS the dram. JPY, KRW and
-- ISK are 0 too; TND, BHD and KWD are 3. Storing the exponent means the code
-- never has to know which is which.
CREATE TABLE currencies (
    code TEXT PRIMARY KEY CHECK (code ~ '^[A-Z]{3}$'), -- ISO 4217

    symbol TEXT NOT NULL,
    -- Where the symbol goes relative to the digits. In the design "$14.00"
    -- but "6,700 ֏", and that difference belongs to the CURRENCY, not to the
    -- reader's language — see the note in frontend/src/lib/currencies.ts for
    -- why this is not left to Intl.NumberFormat's locale rules.
    symbol_position TEXT NOT NULL DEFAULT 'prefix'
                    CHECK (symbol_position IN ('prefix', 'suffix')),

    minor_exponent SMALLINT NOT NULL CHECK (minor_exponent BETWEEN 0 AND 4),

    -- The granularity a converted price is snapped to, in minor units. 1 for
    -- USD (a cent is a real coin); 10 for AMD, because a shelf price of
    -- 5,463 ֏ is not a price anybody writes on a jar. This only ever applies
    -- to CONVERTED prices — an explicit price is whatever the shop typed.
    rounding_step INT NOT NULL DEFAULT 1 CHECK (rounding_step > 0),

    -- The currency every FX rate is quoted against and every price is
    -- authored in. Exactly one row may carry it, enforced below.
    is_base BOOLEAN NOT NULL DEFAULT FALSE,

    is_active  BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INT     NOT NULL DEFAULT 0
);

-- "At most one base currency", as an index. A unique index on a CONSTANT
-- expression over a filtered subset: every row where is_base is true indexes
-- the same key (true), so a second one collides. The singleton form of
-- 000011's one-primary-image index — there the partition was per product,
-- here the whole table is one partition.
CREATE UNIQUE INDEX idx_currencies_one_base ON currencies ((TRUE)) WHERE is_base;

-- Taxonomy, so it is schema meaning and belongs here rather than in the seed
-- — the same argument migration 000008 made for the benefit list.
INSERT INTO currencies (code, symbol, symbol_position, minor_exponent, rounding_step, is_base, sort_order)
VALUES ('USD', '$', 'prefix', 2, 1,  TRUE,  1),
       ('AMD', '֏', 'suffix', 0, 10, FALSE, 2);

-- ── Per-market shelf prices ───────────────────────────────────────────────
--
-- CHOSEN OVER LIVE FX CONVERSION. A shelf price is a business decision, not
-- a derived number: a shop picks a round figure per market and holds it. If
-- every AMD price were computed from a rate, the jar in the window would
-- change price between two page loads, and 6,700 ֏ would render as 6,743 ֏.
--
-- The row is the price. There is no "the" price on the variant any more —
-- product_variants.price_minor is dropped at the bottom of this file.
CREATE TABLE variant_prices (
    variant_id BIGINT NOT NULL REFERENCES product_variants (id) ON DELETE CASCADE,
    -- RESTRICT by default: retiring a currency that still has prices in it
    -- should fail loudly.
    currency    TEXT   NOT NULL REFERENCES currencies (code),
    price_minor BIGINT NOT NULL CHECK (price_minor > 0),
    PRIMARY KEY (variant_id, currency)
);

-- Every existing price was a USD price; make that explicit before the column
-- it lived in disappears.
INSERT INTO variant_prices (variant_id, currency, price_minor)
SELECT v.id, c.code, v.price_minor
FROM product_variants v
CROSS JOIN currencies c
WHERE c.is_base AND v.price_minor > 0;

-- ── Exchange rates ────────────────────────────────────────────────────────
--
-- The FALLBACK for a variant with no explicit price in a currency, and the
-- record that makes an old order reportable. A rate is a fact about a DAY,
-- so as_of is part of the key: rows accumulate, the newest wins, and nothing
-- is ever overwritten. That is what lets "what was this order worth in USD?"
-- be answered next year with the rate that was true at the time, rather than
-- with today's.
CREATE TABLE fx_rates (
    base  TEXT NOT NULL REFERENCES currencies (code),
    quote TEXT NOT NULL REFERENCES currencies (code),
    -- NUMERIC, never float. 1 unit of `base` buys `rate` units of `quote`.
    rate  NUMERIC(18, 8) NOT NULL CHECK (rate > 0),
    as_of DATE           NOT NULL,
    PRIMARY KEY (base, quote, as_of),
    CHECK (base <> quote)
);

-- A bootstrap rate, not a live one: without a row here the fallback below
-- cannot price anything, so the schema would ship in a state where adding a
-- currency silently hides products. Any newer as_of supersedes it.
INSERT INTO fx_rates (base, quote, rate, as_of)
VALUES ('USD', 'AMD', 390.00000000, DATE '2026-08-01');

-- ── What does this variant cost in that currency? ─────────────────────────
--
-- Asked by the catalog list, the count, the facet bounds, the price filter,
-- the price sort, the cart and the checkout. Seven callers, so the answer is
-- defined ONCE, here, as a view — the same instinct that made the catalog's
-- WHERE clause a shared Go constant in store/products.go, applied one layer
-- down where SQL itself can hold the definition.
--
-- A view and not a materialised view: it must be exact the instant a price
-- changes, and it is a join over small tables. It is also a SIMPLE view, so
-- Postgres inlines it and pushes `WHERE currency = $1` down into it rather
-- than building every currency and discarding one.
--
-- The COALESCE is the whole policy in one expression: an explicit shelf
-- price if the shop set one, otherwise the base price converted and snapped
-- to the currency's rounding step.
CREATE VIEW variant_effective_prices AS
SELECT *
FROM (
    SELECT v.id     AS variant_id,
           c.code   AS currency,
           COALESCE(
               vp.price_minor,
               (round(
                    base_price.price_minor::NUMERIC
                    / power(10::NUMERIC, base.minor_exponent) -- → major units, base currency
                    * fx.rate                                 -- → major units, this currency
                    * power(10::NUMERIC, c.minor_exponent)    -- → minor units, this currency
                    / c.rounding_step
                ) * c.rounding_step)::BIGINT
           ) AS price_minor,
           -- Lets a caller tell a shelf price from a computed one. Nothing
           -- reads it yet; it is here because the view is the only place
           -- that knows, and re-deriving it later would mean repeating the
           -- COALESCE.
           vp.price_minor IS NULL AS is_converted
    FROM product_variants v
    CROSS JOIN currencies c
    -- Exactly one row, guaranteed by idx_currencies_one_base — so this join
    -- multiplies nothing.
    JOIN currencies base ON base.is_base
    LEFT JOIN variant_prices vp
           ON vp.variant_id = v.id AND vp.currency = c.code
    LEFT JOIN variant_prices base_price
           ON base_price.variant_id = v.id AND base_price.currency = base.code
    -- The most recent rate at most. LATERAL because the subquery needs
    -- c.code from the row above it, and LIMIT 1 needs its own ORDER BY.
    LEFT JOIN LATERAL (
        SELECT f.rate
        FROM fx_rates f
        WHERE f.base = base.code AND f.quote = c.code
        ORDER BY f.as_of DESC
        LIMIT 1
    ) fx ON TRUE
    WHERE c.is_active
) priced
-- A currency with neither a shelf price nor a rate is not a price of zero,
-- it is an absence. The view emits only rows it can actually price, and
-- callers decide what an absence means: the storefront degrades, checkout
-- refuses.
WHERE price_minor IS NOT NULL;

-- ── An order is charged in ONE currency ───────────────────────────────────
--
-- Decision #3 (snapshot what the customer saw) extended one step: the
-- price snapshots in order_items are meaningless without knowing which
-- currency they are denominated in.
ALTER TABLE orders
    ADD COLUMN currency TEXT NOT NULL DEFAULT 'USD' REFERENCES currencies (code),
    -- How many units of `currency` one unit of the base currency bought at
    -- checkout. Reporting only — it is NOT what the customer was charged,
    -- which is the sum of the snapshots in order_items. NULL when the order
    -- was in the base currency, or when no rate was on file: an explicit
    -- "not applicable" rather than a decorative 1.0.
    ADD COLUMN fx_rate_used NUMERIC(18, 8) CHECK (fx_rate_used > 0);

-- The DEFAULT existed only to backfill the rows already in the table. Left
-- in place it would let a future INSERT that forgot the column quietly
-- charge in dollars; dropping it makes that a NOT NULL violation instead.
ALTER TABLE orders ALTER COLUMN currency DROP DEFAULT;

-- ── Retire the single-currency column ─────────────────────────────────────
--
-- Kept nowhere, not even "for now": a copy of the USD price on the variant
-- and another in variant_prices would be two sources of truth for one
-- number, and they would drift the first time someone updated only one.
ALTER TABLE product_variants DROP COLUMN price_minor;
