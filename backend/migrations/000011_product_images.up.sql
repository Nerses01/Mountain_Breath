-- The product page's gallery: one hero plus four thumbnails
-- (docs/PLAN_ERA_2.md, phase E3).
--
-- This is the FIRST table where decision #6 and decision #4 disagree about
-- shape, and the difference is worth being explicit about. A highlight bullet
-- is entirely text, so E3 keys those rows by locale directly. An IMAGE is
-- mostly not text: the file, its position and whether it is the hero mean the
-- same thing in every language, and only `alt` is prose. So this follows #6's
-- original split — locale-invariant columns stay on the parent, the one
-- translatable field moves out.

CREATE TABLE product_images (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id BIGINT  NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    url        TEXT    NOT NULL,
    sort_order INT     NOT NULL DEFAULT 0,
    -- The gallery's hero. A flag rather than products.primary_image_id
    -- because the FK would point the other way and make "delete this image"
    -- fail against the product still referencing it.
    is_primary BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_product_images_product_id ON product_images (product_id, sort_order);

-- ONE primary image per product, enforced by the database rather than by
-- application code that remembers to clear the old flag.
--
-- A plain UNIQUE (product_id, is_primary) would be wrong: it would also
-- forbid a product having two NON-primary images, which is the normal case.
-- A PARTIAL index — one with a WHERE clause — indexes only the rows matching
-- it, so uniqueness applies exclusively to the rows where is_primary is true.
-- Postgres has no "unique among some rows" constraint syntax; this index IS
-- the constraint.
--
-- Consequence worth knowing before writing the store: setting a new primary
-- must clear the old one in the SAME statement or transaction, because the
-- index rejects the intermediate state where two rows are true.
CREATE UNIQUE INDEX idx_product_images_one_primary
    ON product_images (product_id)
    WHERE is_primary;

-- Alt text is not decoration: a screen reader reads it aloud, in the
-- visitor's language, and it is the only description a non-sighted customer
-- gets of the product photo. Same table shape as category_translations.
CREATE TABLE product_image_translations (
    image_id BIGINT NOT NULL REFERENCES product_images (id) ON DELETE CASCADE,
    locale   TEXT   NOT NULL CHECK (locale IN ('en', 'hy', 'ru')),
    alt      TEXT   NOT NULL,
    PRIMARY KEY (image_id, locale)
);

-- Backfill: every product that already has an uploaded image gets it as its
-- primary gallery image, so nothing disappears from the storefront the moment
-- the read path switches over.
--
-- products.image_url is deliberately LEFT IN PLACE — the admin upload
-- endpoint still writes it, and dropping the column before the new write path
-- exists would take the admin's image upload with it. It goes in migration
-- 000015, after the gallery is writable, which is the same
-- add-backfill-then-drop sequence 000007 used for the name columns.
INSERT INTO product_images (product_id, url, sort_order, is_primary)
SELECT id, image_url, 0, TRUE
FROM products
WHERE image_url <> '';

-- The alt text for a backfilled image is the product's own name in each
-- language: no better description exists, and "" would be worse than
-- imperfect — an empty alt tells a screen reader the image is decorative,
-- which a product photo is not.
INSERT INTO product_image_translations (image_id, locale, alt)
SELECT i.id, t.locale, t.name
FROM product_images i
JOIN product_translations t ON t.product_id = i.product_id;
