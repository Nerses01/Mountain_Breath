-- One short product video, stored as a row in product_images rather than in
-- a table of its own. The row shape barely differs — a URL on disk, owned by
-- a product, with a per-locale accessible label in the translations table —
-- and reusing the table means the existing admin DELETE endpoint, the alt
-- machinery and the cascade all apply to the video with no new code. What
-- DOES differ is captured by constraints below, not by convention.
--
-- The discriminator column is the SQL cousin of a tagged union: one table,
-- a `kind` tag, and CHECK constraints expressing which states are legal per
-- tag — where C++ would give each alternative its own type inside a
-- std::variant, SQL keeps one row shape and constrains it.
ALTER TABLE product_images
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'image'
        CONSTRAINT product_images_kind_check CHECK (kind IN ('image', 'video'));

-- The gallery's hero is always a PHOTO. Cards and the gallery's top slot
-- render an <img>; a video row with is_primary would make both draw nothing.
-- A row-local CHECK is enough — it needs no look at other rows.
ALTER TABLE product_images
    ADD CONSTRAINT product_images_video_not_primary
        CHECK (NOT (kind = 'video' AND is_primary));

-- At most ONE video per product — the same partial-unique-index trick as
-- idx_product_images_one_primary in 000011: only rows matching the WHERE are
-- indexed, so uniqueness binds exclusively the video rows.
CREATE UNIQUE INDEX idx_product_images_one_video
    ON product_images (product_id)
    WHERE kind = 'video';

-- Deliberately NO constraint for the three-image cap. A per-group count
-- limit cannot be a CHECK (those see one row) or a unique index (nothing is
-- duplicated); it would need a trigger. The cap is a business rule about NEW
-- uploads — galleries that already exceed it stay legal — so it is enforced
-- in the store's AddProductImage transaction instead, under a row lock.
