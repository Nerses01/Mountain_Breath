DROP INDEX IF EXISTS idx_products_rating;

ALTER TABLE products
    DROP COLUMN IF EXISTS rating_avg,
    DROP COLUMN IF EXISTS rating_count;

-- Indexes go with the table.
DROP TABLE IF EXISTS reviews;
