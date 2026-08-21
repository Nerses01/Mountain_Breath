-- Restore the column exactly as 000002 declared it…
ALTER TABLE products ADD COLUMN image_url TEXT NOT NULL DEFAULT '';

-- …and refill it from the gallery hero, inverting 000011's backfill. The
-- ORDER BY mirrors the read path: the flagged hero first, else the first
-- photo in gallery order. Only photos — a video URL in an <img> would be a
-- worse restoration than an empty string.
UPDATE products p
SET image_url = COALESCE(
    (SELECT i.url
     FROM product_images i
     WHERE i.product_id = p.id AND i.kind = 'image'
     ORDER BY i.is_primary DESC, i.sort_order, i.id
     LIMIT 1),
    '');
