-- The product page's meta row (Harvest / Shipping / Lab report), the
-- disclaimer under "What it does", and the Storage tab
-- (docs/PLAN_ERA_2.md, phase E3).
--
-- Split by the same question every field in this project gets asked: does it
-- mean the same thing in three languages?

-- Locale-INVARIANT, so they stay on the product row.
ALTER TABLE products
    -- "RJ-0626" — a batch identifier, not prose. Reads the same in Yerevan
    -- and in Moscow, exactly like sku.
    ADD COLUMN lab_batch TEXT NOT NULL DEFAULT '',
    -- A fact about the product, not a sentence about it. The frontend turns
    -- it into whatever the reader's language calls a cold chain, and E6 will
    -- want it as a BOOLEAN when it charges chilled shipping — a translated
    -- string could not be reasoned about.
    ADD COLUMN is_cold_chain BOOLEAN NOT NULL DEFAULT FALSE;

-- TRANSLATABLE, so they join the rest of the product's prose.
--
-- New columns on product_translations rather than four more tables: these are
-- scalar fields of the product, not ordered collections, so they belong
-- exactly where name and description already are. The table's generated
-- search_tsv is untouched — nobody searches the catalog by disclaimer, and
-- widening the vector would dilute the ranking of the fields people do
-- search by.
ALTER TABLE product_translations
    -- "Not a medicine. Avoid if you are allergic to bee products."
    ADD COLUMN disclaimer    TEXT NOT NULL DEFAULT '',
    -- The Storage tab's paragraph.
    ADD COLUMN storage_note  TEXT NOT NULL DEFAULT '',
    -- "June 2026, Hive 41". Free text, and deliberately so: it could be
    -- modelled as a date plus a hive number, which would be locale-invariant
    -- and formatted per language for free. That is the better shape for a
    -- catalog of thousands — and the wrong one for six products a family
    -- writes by hand, where it would trade a text box for a date picker and
    -- a numeric field to save two translations.
    ADD COLUMN harvest_note  TEXT NOT NULL DEFAULT '',
    -- "Chilled, 2–4 days".
    ADD COLUMN shipping_note TEXT NOT NULL DEFAULT '';
