DROP INDEX idx_product_images_one_video;

-- A video row cannot survive the column drop: without `kind` it would be
-- indistinguishable from a photo, and every card and gallery would try to
-- render an .mp4 in an <img>. Deleting is the honest rollback — the
-- translations rows follow via ON DELETE CASCADE.
DELETE FROM product_images WHERE kind = 'video';

ALTER TABLE product_images
    DROP CONSTRAINT product_images_video_not_primary;

-- Dropping the column takes its inline CHECK (product_images_kind_check)
-- down with it.
ALTER TABLE product_images
    DROP COLUMN kind;
