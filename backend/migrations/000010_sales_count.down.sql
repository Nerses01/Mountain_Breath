-- Dropping the column drops its index and CHECK constraint with it; the
-- explicit DROP INDEX is there so the down migration reads as the exact
-- inverse of the up.
DROP INDEX IF EXISTS idx_products_sales_count;

ALTER TABLE products
    DROP COLUMN IF EXISTS sales_count;
