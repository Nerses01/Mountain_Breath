-- Translatable content for the three shop languages (docs/PLAN_ERA_2.md,
-- phase E1.5; decision #15 in ARCHITECTURE.md).
--
-- Only human-language text moves out of the parent row. Locale-invariant
-- fields — slug, sku, price_minor, stock_qty, sort_order — stay put, because
-- they mean the same thing in every language and duplicating them per locale
-- would just create three ways to disagree.
--
-- The locale CHECK hardcodes the set rather than referencing a `locales`
-- table. Adding a fourth language is a migration either way: it also needs
-- message catalogues, a font subset with the right glyphs, and a text search
-- configuration. A constraint that fails loudly on an unknown locale is
-- worth more here than the flexibility of an INSERT.

CREATE TABLE category_translations (
    category_id BIGINT NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    locale      TEXT   NOT NULL CHECK (locale IN ('en', 'hy', 'ru')),
    name        TEXT   NOT NULL,
    -- Composite PK: one name per category per language. Its leading column
    -- is category_id, so this also serves as the FK index — no separate
    -- CREATE INDEX needed (unlike products.category_id in 000002).
    PRIMARY KEY (category_id, locale)
);

CREATE TABLE product_translations (
    product_id  BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    locale      TEXT   NOT NULL CHECK (locale IN ('en', 'hy', 'ru')),
    name        TEXT   NOT NULL,
    description TEXT   NOT NULL DEFAULT '',

    -- Per-locale full-text search, the multilingual version of 000005.
    --
    -- A generated column must be IMMUTABLE, which rules out the obvious
    -- `to_tsvector(locale::regconfig, name)` — casting text to regconfig
    -- reads the catalog and is only STABLE. A CASE over LITERAL config
    -- names is immutable, though, because each branch is a constant, and
    -- that is enough to get real per-language stemming:
    --   'Wildflower'  -> 'wildflow'   (english)
    --   'цветочный'   -> 'цветочн'    (russian, and ё normalises to е)
    --   'Լեռնային'    -> 'լեռնայ'     (armenian)
    -- Postgres ships all three configurations; `SELECT cfgname FROM
    -- pg_ts_config` lists 29 of them.
    search_tsv tsvector GENERATED ALWAYS AS (
        CASE locale
            WHEN 'hy' THEN
                setweight(to_tsvector('armenian', coalesce(name, '')), 'A') ||
                setweight(to_tsvector('armenian', coalesce(description, '')), 'B')
            WHEN 'ru' THEN
                setweight(to_tsvector('russian', coalesce(name, '')), 'A') ||
                setweight(to_tsvector('russian', coalesce(description, '')), 'B')
            ELSE
                setweight(to_tsvector('english', coalesce(name, '')), 'A') ||
                setweight(to_tsvector('english', coalesce(description, '')), 'B')
        END
    ) STORED,

    PRIMARY KEY (product_id, locale)
);

CREATE INDEX idx_product_translations_search
    ON product_translations USING GIN (search_tsv);

-- The trigram half of search v2 (000006), now per translation: prefix,
-- substring and typo tolerance work the same way in every language because
-- trigrams do not care what the letters mean.
CREATE INDEX idx_product_translations_name_trgm
    ON product_translations USING GIN (name gin_trgm_ops);

-- Backfill: every existing row's text becomes its English translation.
-- `products.name`/`description` and `categories.name` are deliberately LEFT
-- IN PLACE — products.search_tsv is a generated column reading them and
-- idx_products_name_trgm indexes them, so dropping the columns here would
-- take Era I's whole search implementation with it while the store layer
-- still depends on it. They are dropped in a follow-up migration once the
-- store reads translations, the same add-backfill-then-drop sequence the
-- plan uses for products.image_url.
INSERT INTO category_translations (category_id, locale, name)
SELECT id, 'en', name FROM categories;

INSERT INTO product_translations (product_id, locale, name, description)
SELECT id, 'en', name, description FROM products;
