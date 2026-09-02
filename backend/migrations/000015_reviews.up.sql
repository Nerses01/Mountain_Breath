-- Customer reviews and the rating aggregate the catalog sorts and renders by
-- (docs/PLAN_ERA_2.md, phase E4).

CREATE TABLE reviews (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    -- RESTRICT, matching orders.user_id: a review is a statement by a named
    -- person, and deleting the person out from under it would leave the shop
    -- displaying an opinion nobody is attached to.
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE RESTRICT,

    rating INT  NOT NULL CHECK (rating BETWEEN 1 AND 5),
    title  TEXT NOT NULL DEFAULT '',
    body   TEXT NOT NULL DEFAULT '',

    -- Nothing is public until a human says so. `pending` is the default
    -- rather than `published` so that forgetting to moderate fails CLOSED —
    -- the worst outcome of a bug here is an unpublished review, not an
    -- unmoderated one on the storefront.
    status TEXT NOT NULL DEFAULT 'pending'
           CHECK (status IN ('pending', 'published', 'rejected')),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One review per person per product. This is the cheapest anti-spam
    -- measure there is, and it is a CONSTRAINT rather than a check in the
    -- handler because two concurrent submissions would both pass an
    -- application-level "have you reviewed this already?" test.
    UNIQUE (product_id, user_id)
);

-- The public read: one product's published reviews, newest first. A partial
-- index, like 000011's — the storefront never asks for pending or rejected
-- rows, so there is no reason to index them for this query.
CREATE INDEX idx_reviews_published
    ON reviews (product_id, created_at DESC)
    WHERE status = 'published';

-- The moderation queue reads the other way: by status, across all products.
CREATE INDEX idx_reviews_status ON reviews (status, created_at DESC);

-- ── The aggregate ─────────────────────────────────────────────────────────
--
-- Denormalized for the same reason sales_count was in 000010: the LIST query
-- needs it. A card shows ★★★★☆ (12), and computing that per product means
-- aggregating the reviews table on every catalog page — the work grows with
-- every review ever written, to render twelve cards.
--
-- WHY A STORED AVERAGE AND NOT A RUNNING SUM. The tempting alternative is
-- `rating_sum` + `rating_count`, with the average derived on read: it is
-- exact by construction and updates incrementally. It also cannot be indexed
-- for `ORDER BY rating_avg`, which is the whole point of storing it. So the
-- average is stored, and kept honest by RECOMPUTING it from the reviews
-- table inside the same transaction as any change — never by nudging the old
-- value, which is how floating-point aggregates drift out of agreement with
-- the rows they claim to summarise.
--
-- NUMERIC, not float: 4.65 is a number a human reads, and binary floating
-- point cannot represent it exactly. Same reasoning as money, one step down
-- in severity.
ALTER TABLE products
    ADD COLUMN rating_avg   NUMERIC(3, 2) NOT NULL DEFAULT 0
        CHECK (rating_avg >= 0 AND rating_avg <= 5),
    ADD COLUMN rating_count INT           NOT NULL DEFAULT 0
        CHECK (rating_count >= 0);

-- Sorting by rating reads this on every Shop page load that asks for it.
-- rating_count rides along so the index also serves "well-rated AND actually
-- reviewed", which is the only honest way to rank by stars.
CREATE INDEX idx_products_rating ON products (rating_avg DESC, rating_count DESC);
