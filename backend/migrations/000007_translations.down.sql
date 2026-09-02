-- Dropping the tables takes their indexes and the generated search column
-- with them. No data is lost that the parent rows do not still hold: this
-- migration's up only ever backfilled English text copied FROM
-- categories.name / products.name / products.description, which are still
-- there. Armenian and Russian rows added after the fact WOULD be lost —
-- which is exactly what "down" means, and why it is a development tool.
DROP TABLE IF EXISTS product_translations;
DROP TABLE IF EXISTS category_translations;
