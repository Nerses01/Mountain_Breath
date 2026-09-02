-- "Often taken together" (docs/PLAN_ERA_2.md, phase E3): a curated list of
-- products to show under a product, with a computed fallback when the admin
-- has not curated one.
--
-- A self-referencing many-to-many — the same table on both sides of the join
-- — which brings two failure modes a normal join table does not have, both
-- worth stating in SQL rather than trusting the admin form to prevent:
--
--   1. A product related to ITSELF. Harmless-looking, and it would render
--      the product you are already reading in its own "often taken together"
--      row. The CHECK forbids it.
--   2. The relation is NOT symmetric here. Honey → propolis does not imply
--      propolis → honey, because the merchandising reason may only run one
--      way ("stir this into honey" belongs under the jelly, not the honey).
--      That is a deliberate choice, recorded so the missing reverse row
--      reads as intent rather than as a bug.
CREATE TABLE product_related (
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    -- CASCADE on both sides: if either product goes, the pairing is
    -- meaningless. Nothing else references this row, so nothing is orphaned.
    related_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    sort_order INT    NOT NULL DEFAULT 0,
    PRIMARY KEY (product_id, related_id),
    CHECK (product_id <> related_id)
);

-- The PK leads with product_id, which is the direction every read goes
-- ("what goes with this product"). No mirror index here, unlike
-- product_benefits in 000008: nothing asks "which products point AT this
-- one", because the relation is directed and the reverse question has no
-- page to render it on.
