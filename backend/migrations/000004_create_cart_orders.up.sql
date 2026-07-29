-- The cart is 1:1 with the user, so no separate carts table is needed:
-- a cart IS the set of cart_items rows for a user.
CREATE TABLE cart_items (
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    variant_id BIGINT      NOT NULL REFERENCES product_variants (id) ON DELETE CASCADE,
    qty        INT         NOT NULL CHECK (qty > 0),
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- composite primary key: one row per user+variant; adding the same
    -- variant twice is an update, not a duplicate
    PRIMARY KEY (user_id, variant_id)
);

CREATE TABLE orders (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- RESTRICT: users with order history cannot be hard-deleted (audit trail)
    user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    status      TEXT        NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered', 'cancelled')),
    total_minor BIGINT      NOT NULL CHECK (total_minor >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_orders_user_id ON orders (user_id);

CREATE TABLE order_items (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id             BIGINT NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    -- RESTRICT: a variant referenced by an order must not disappear
    variant_id           BIGINT NOT NULL REFERENCES product_variants (id) ON DELETE RESTRICT,
    -- snapshots: the order shows what the customer actually bought, even if
    -- the product is later renamed or repriced
    name_snapshot        TEXT   NOT NULL,
    label_snapshot       TEXT   NOT NULL,
    price_minor_snapshot BIGINT NOT NULL,
    qty                  INT    NOT NULL CHECK (qty > 0)
);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);
