-- The product page's editorial content (docs/PLAN_ERA_2.md, phase E3):
-- the "What it does" bullets, and the three Morning / Course / Pairs-with
-- cards under the tabs.
--
-- DECISION #4, decided 2026-08-10: these rows are keyed BY LOCALE, not split
-- into a parent row plus a translation table.
--
-- The reason is that decision #6's split exists to keep locale-invariant
-- fields out of the per-language rows — a slug, an SKU, a price. A highlight
-- bullet has no such field. Splitting it would produce a parent row carrying
-- nothing but `sort_order` and an id, plus a translation table holding the
-- only real content, and every read would join the two back together to
-- recover what one row already said. So #6's PRINCIPLE (don't duplicate
-- locale-invariant facts per language) is honoured here by a different shape,
-- because there is nothing locale-invariant left to protect.
--
-- What that buys, beyond two tables instead of four: the languages are free
-- to differ in COUNT. Armenian may need four bullets where English needs
-- three, and a translator is not forced to pad. What it costs: no bullet is
-- formally "the translation of" another, so the three languages can drift in
-- order as well as in number. For editorial prose that is the right trade;
-- for a product NAME it would not be, which is why names still live in
-- product_translations.

CREATE TABLE product_highlights (
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    locale     TEXT   NOT NULL CHECK (locale IN ('en', 'hy', 'ru')),
    sort_order INT    NOT NULL,
    text       TEXT   NOT NULL,
    -- The PK is the whole identity: one bullet per product per language per
    -- position. Its leading column is product_id, so it also serves as the
    -- FK index and as the ordering for a read filtered to one locale.
    PRIMARY KEY (product_id, locale, sort_order)
);

-- "Morning / A grain of rice / Under the tongue before breakfast…"
--
-- kicker, title and body are three fields rather than one blob because the
-- design gives each its own type treatment, and a single text column would
-- push the split into the frontend where it could not be validated.
CREATE TABLE product_usage_cards (
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    locale     TEXT   NOT NULL CHECK (locale IN ('en', 'hy', 'ru')),
    sort_order INT    NOT NULL,
    kicker     TEXT   NOT NULL,
    title      TEXT   NOT NULL,
    body       TEXT   NOT NULL,
    PRIMARY KEY (product_id, locale, sort_order)
);
