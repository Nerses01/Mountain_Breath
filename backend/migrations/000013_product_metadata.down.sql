ALTER TABLE product_translations
    DROP COLUMN IF EXISTS disclaimer,
    DROP COLUMN IF EXISTS storage_note,
    DROP COLUMN IF EXISTS harvest_note,
    DROP COLUMN IF EXISTS shipping_note;

ALTER TABLE products
    DROP COLUMN IF EXISTS lab_batch,
    DROP COLUMN IF EXISTS is_cold_chain;
