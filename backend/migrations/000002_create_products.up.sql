CREATE TABLE products (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- RESTRICT: deleting a category with products must fail loudly,
    -- not silently take the products with it.
    category_id BIGINT      NOT NULL REFERENCES categories (id) ON DELETE RESTRICT,
    slug        TEXT        NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    image_url   TEXT        NOT NULL DEFAULT '',
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Postgres does NOT auto-index foreign key columns; without this,
-- "products in category X" would scan the whole table.
CREATE INDEX idx_products_category_id ON products (category_id);

CREATE TABLE product_variants (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- CASCADE: a variant has no meaning without its product.
    product_id  BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    sku         TEXT   NOT NULL UNIQUE,
    label       TEXT   NOT NULL, -- "250 g", "0.5 L"
    price_minor BIGINT NOT NULL CHECK (price_minor >= 0),
    stock_qty   INT    NOT NULL DEFAULT 0 CHECK (stock_qty >= 0),
    -- one product can't have two variants with the same label
    UNIQUE (product_id, label)
);

CREATE INDEX idx_product_variants_product_id ON product_variants (product_id);
