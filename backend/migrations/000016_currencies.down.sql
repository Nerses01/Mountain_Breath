-- Put the single-currency world back, base-currency prices and all.

DROP VIEW IF EXISTS variant_effective_prices;

-- Re-add nullable, backfill from the base currency's rows, THEN constrain.
-- Adding it NOT NULL in one step would need a DEFAULT, and a default price
-- is a worse lie than a brief nullable column.
ALTER TABLE product_variants ADD COLUMN price_minor BIGINT;

UPDATE product_variants v
SET price_minor = vp.price_minor
FROM variant_prices vp
JOIN currencies c ON c.code = vp.currency AND c.is_base
WHERE vp.variant_id = v.id;

-- A variant priced in AMD only cannot round-trip through a USD-shaped
-- column. Zero is the honest marker for "this price did not survive the
-- downgrade" — the original CHECK allowed it.
UPDATE product_variants SET price_minor = 0 WHERE price_minor IS NULL;

ALTER TABLE product_variants
    ALTER COLUMN price_minor SET NOT NULL,
    ADD CONSTRAINT product_variants_price_minor_check CHECK (price_minor >= 0);

-- Drop the referencing columns/tables before the table they reference.
ALTER TABLE orders
    DROP COLUMN IF EXISTS currency,
    DROP COLUMN IF EXISTS fx_rate_used;

DROP TABLE IF EXISTS fx_rates;
DROP TABLE IF EXISTS variant_prices;
-- Index goes with the table.
DROP TABLE IF EXISTS currencies;
