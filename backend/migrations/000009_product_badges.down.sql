-- Dropping a column drops its CHECK constraint with it.
ALTER TABLE products
    DROP COLUMN IF EXISTS badge,
    DROP COLUMN IF EXISTS badge_tone;
