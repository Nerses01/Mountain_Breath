-- The "Good for" taxonomy behind the Shop sidebar's second facet
-- (docs/PLAN_ERA_2.md, phase E2).
--
-- Categories and benefits look similar but are shaped differently, and the
-- difference is the point of this migration. A product has exactly ONE
-- category (products.category_id, a foreign key) — honey is not also beeswax.
-- A product has ANY NUMBER of benefits: royal jelly is good for energy and
-- for skin. One-to-many needs a column; many-to-many needs a table of its own.

CREATE TABLE benefits (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug       TEXT NOT NULL UNIQUE,
    -- Display order in the sidebar. The design draws the chips in a fixed
    -- order (Energy first), and alphabetical would be a different order in
    -- each of the three languages.
    sort_order INT  NOT NULL DEFAULT 0
);

-- The join table. Its primary key is the PAIR, which is what makes the
-- relationship a set: (honey, energy) can only be stated once, so no amount
-- of double-clicking in the admin form can make one product count twice in
-- a facet total.
CREATE TABLE product_benefits (
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    benefit_id BIGINT NOT NULL REFERENCES benefits (id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, benefit_id)
);

-- The PK index above is ordered (product_id, benefit_id), so it answers
-- "which benefits does this product have" but not "which products have this
-- benefit" — a composite index only helps queries that constrain its LEADING
-- column. The facet query filters by benefit slug, so it needs the mirror.
CREATE INDEX idx_product_benefits_benefit_id ON product_benefits (benefit_id);

-- Same shape as category_translations in 000007: only human-language text
-- moves out of the parent row, the slug and sort_order stay put. No
-- search_tsv here — benefits are filter chips, not search targets.
CREATE TABLE benefit_translations (
    benefit_id BIGINT NOT NULL REFERENCES benefits (id) ON DELETE CASCADE,
    locale     TEXT   NOT NULL CHECK (locale IN ('en', 'hy', 'ru')),
    name       TEXT   NOT NULL,
    PRIMARY KEY (benefit_id, locale)
);

-- The five chips the design draws, seeded here rather than in seed.sql.
-- seed.sql is development sample data that a real deployment throws away;
-- this taxonomy is part of the schema's meaning — products.benefit rows in
-- E2's seed reference these slugs, and the sidebar renders exactly this set.
INSERT INTO benefits (slug, sort_order) VALUES
    ('energy',     1),
    ('immunity',   2),
    ('skin',       3),
    ('recovery',   4),
    ('sweetening', 5);

INSERT INTO benefit_translations (benefit_id, locale, name)
SELECT b.id, v.locale, v.name
FROM (VALUES
    ('energy',     'en', 'Energy'),
    ('energy',     'hy', 'Էներգիա'),
    ('energy',     'ru', 'Энергия'),
    ('immunity',   'en', 'Immunity'),
    ('immunity',   'hy', 'Իմունիտետ'),
    ('immunity',   'ru', 'Иммунитет'),
    ('skin',       'en', 'Skin'),
    ('skin',       'hy', 'Մաշկ'),
    ('skin',       'ru', 'Кожа'),
    ('recovery',   'en', 'Recovery'),
    ('recovery',   'hy', 'Վերականգնում'),
    ('recovery',   'ru', 'Восстановление'),
    ('sweetening', 'en', 'Sweetening'),
    ('sweetening', 'hy', 'Քաղցրացում'),
    ('sweetening', 'ru', 'Подслащивание')
) AS v(slug, locale, name)
JOIN benefits b ON b.slug = v.slug;
