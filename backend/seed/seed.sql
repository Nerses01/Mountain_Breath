-- Development seed data: the six hive products of the design
-- (docs/PLAN_ERA_2.md §1.1, decision #1 "catalog scope — apiary-only").
--
-- Run with:
--   Get-Content backend\seed\seed.sql | docker exec -i mb-postgres psql -U mb -d mountain_breath
--
-- IDEMPOTENT AND CONVERGENT. Era I's seed used ON CONFLICT DO NOTHING, which
-- makes re-running safe but not correct: a row that already exists keeps its
-- OLD content, so editing this file changed nothing on a database that had
-- been seeded before. Every upsert below is DO UPDATE instead, so running it
-- twice and running it after an edit both leave the database in the state
-- this file describes.
--
-- Prices are USD minor units — $14.00 = 1400 (decision: E2 seeds one
-- currency, E5 introduces variant_prices and backfills AMD). They are
-- PLACEHOLDERS from the mock, not the family's real shelf prices (§1.4).

-- ── Era I's sample catalog ────────────────────────────────────────────────
-- Tea and coffee are gone from the catalog, so they should go from a
-- database that was seeded before E2 as well — otherwise the sidebar counts
-- nine categories and the design's "6 products" never matches.
--
-- Deletes that cannot succeed are skipped rather than allowed to fail:
-- order_items.variant_id is ON DELETE RESTRICT, so a product someone has
-- actually ordered in a dev session refuses to disappear. Those are
-- deactivated instead, which removes them from every storefront query while
-- leaving the order history readable. Everything else cascades away through
-- product_variants → cart_items.
UPDATE products SET is_active = FALSE
WHERE slug IN ('mountain-herbal-tea', 'wild-thyme-tea', 'armenian-coffee', 'wildflower-honey');

DELETE FROM products p
WHERE p.slug IN ('mountain-herbal-tea', 'wild-thyme-tea', 'armenian-coffee', 'wildflower-honey')
  AND NOT EXISTS (
      SELECT 1 FROM order_items oi
      JOIN product_variants v ON v.id = oi.variant_id
      WHERE v.product_id = p.id);

DELETE FROM categories c
WHERE c.slug IN ('herbal-tea', 'coffee')
  AND NOT EXISTS (SELECT 1 FROM products p WHERE p.category_id = c.id);

-- ── Categories ────────────────────────────────────────────────────────────
-- One per product: the design's sidebar reads Honey / Beeswax / Propolis /
-- Royal jelly / Bee pollen / Bee venom with a count of 1 each. 'honey'
-- already exists from Era I as "Honey & Sweets" and is renamed by the
-- upsert rather than deleted, so any product still pointing at it survives.
INSERT INTO categories (slug, name, sort_order) VALUES
    ('honey',       'Honey',       1),
    ('beeswax',     'Beeswax',     2),
    ('propolis',    'Propolis',    3),
    ('royal-jelly', 'Royal jelly', 4),
    ('bee-pollen',  'Bee pollen',  5),
    ('bee-venom',   'Bee venom',   6)
ON CONFLICT (slug) DO UPDATE
    SET name = EXCLUDED.name, sort_order = EXCLUDED.sort_order;

-- Armenian and Russian are MACHINE-ASSISTED AND FLAGGED FOR NATIVE REVIEW,
-- the same caveat E1.5 attached to the message catalogues. Apiary vocabulary
-- is specialist: "royal jelly" is մայրական կաթ / маточное молочко, literally
-- "mother's milk", and a plausible-looking wrong word here would be
-- invisible to anyone who does not keep bees.
INSERT INTO category_translations (category_id, locale, name)
SELECT c.id, v.locale, v.name
FROM (VALUES
    ('honey',       'en', 'Honey'),
    ('honey',       'hy', 'Մեղր'),
    ('honey',       'ru', 'Мёд'),
    ('beeswax',     'en', 'Beeswax'),
    ('beeswax',     'hy', 'Մեղրամոմ'),
    ('beeswax',     'ru', 'Пчелиный воск'),
    ('propolis',    'en', 'Propolis'),
    ('propolis',    'hy', 'Պրոպոլիս'),
    ('propolis',    'ru', 'Прополис'),
    ('royal-jelly', 'en', 'Royal jelly'),
    ('royal-jelly', 'hy', 'Մայրական կաթ'),
    ('royal-jelly', 'ru', 'Маточное молочко'),
    ('bee-pollen',  'en', 'Bee pollen'),
    ('bee-pollen',  'hy', 'Ծաղկափոշի'),
    ('bee-pollen',  'ru', 'Пчелиная пыльца'),
    ('bee-venom',   'en', 'Bee venom'),
    ('bee-venom',   'hy', 'Մեղվի թույն'),
    ('bee-venom',   'ru', 'Пчелиный яд')
) AS v(cat_slug, locale, name)
JOIN categories c ON c.slug = v.cat_slug
ON CONFLICT (category_id, locale) DO UPDATE SET name = EXCLUDED.name;

-- ── Products ──────────────────────────────────────────────────────────────
-- badge is a KEY, not a sentence (migration 000009) — the three message
-- catalogues own the wording. badge_tone is presentation: everything is the
-- design's honey chip except the cold-chain badge, which Badge.tsx already
-- called out as looking better dark.
--
-- sales_count is dev data, unlike the column itself: seeded with a spread so
-- the "Most loved" sort is visibly not alphabetical order. In production the
-- checkout transaction is the only writer.
INSERT INTO products (category_id, slug, name, description, badge, badge_tone, sales_count)
SELECT c.id, v.slug, v.name, v.description, v.badge, v.badge_tone, v.sales_count
FROM (VALUES
    ('honey', 'mountain-wildflower-honey', 'Mountain Wildflower Honey',
     'Sweet liquid made from flower nectar, used as food and a natural sweetener. Poured into 500 g and 1 kg jars the same week it is spun.',
     'best_seller', 'honey', 148),
    ('beeswax', 'pure-beeswax-blocks', 'Pure Beeswax Blocks',
     'Oily wax secreted from bee glands to build honeycombs, used in candles and skin creams. Cast into 100 g blocks and boxed in fours.',
     'for_makers', 'honey', 42),
    ('propolis', 'raw-propolis-tincture', 'Raw Propolis Tincture',
     'Resinous bee glue collected from plants to seal and protect the hive, known for antimicrobial properties. Bottled in a dropper flask.',
     'immunity', 'honey', 96),
    ('royal-jelly', 'fresh-royal-jelly', 'Fresh Royal Jelly',
     'Milky fluid fed to queen larvae, used in health supplements and cosmetics. Kept chilled in a small dark jar from the hive to your door.',
     'cold_chain', 'dark', 61),
    ('bee-pollen', 'bee-pollen-granules', 'Bee Pollen Granules',
     'Flower pollen packed by bees, rich in protein and nutrients. Dried gently and sealed in a resealable pouch.',
     'protein', 'honey', 74),
    ('bee-venom', 'bee-venom-serum', 'Bee Venom Serum',
     'Secreted defense fluid sometimes used in alternative therapies. Blended into a light serum in a small glass bottle.',
     'new', 'honey', 12)
) AS v(cat_slug, slug, name, description, badge, badge_tone, sales_count)
JOIN categories c ON c.slug = v.cat_slug
ON CONFLICT (slug) DO UPDATE
    SET category_id  = EXCLUDED.category_id,
        name         = EXCLUDED.name,
        description  = EXCLUDED.description,
        badge        = EXCLUDED.badge,
        badge_tone   = EXCLUDED.badge_tone,
        sales_count  = EXCLUDED.sales_count,
        -- Undo the deactivation above if a slug was reused.
        is_active    = TRUE;

-- The English rows are written explicitly, not left to the fallback chain:
-- product_translations.search_tsv is stemmed per locale, so an English row
-- here is what makes an English full-text search rank this product properly
-- rather than reaching two levels down to products.search_tsv.
--
-- The container word ("jar", "dropper", "pouch") lives in this copy, because
-- variant labels are pure measurements — the E2 decision that the label is
-- locale-invariant like sku and price_minor, so "500 g" needs no translation
-- while "500 g jar" would.
INSERT INTO product_translations (product_id, locale, name, description)
SELECT p.id, v.locale, v.name, v.description
FROM (VALUES
    ('mountain-wildflower-honey', 'en', 'Mountain Wildflower Honey',
     'Sweet liquid made from flower nectar, used as food and a natural sweetener. Poured into 500 g and 1 kg jars the same week it is spun.'),
    ('mountain-wildflower-honey', 'hy', 'Լեռնային վայրի ծաղիկների մեղր',
     'Ծաղկի նեկտարից ստացված քաղցր հեղուկ՝ և՛ սնունդ, և՛ բնական քաղցրացուցիչ։ Լցվում է 500 գ և 1 կգ բանկաների մեջ մզման նույն շաբաթում։'),
    ('mountain-wildflower-honey', 'ru', 'Горный цветочный мёд',
     'Сладкая жидкость из цветочного нектара — и еда, и натуральный подсластитель. Разливаем в банки по 500 г и 1 кг на той же неделе, когда качаем.'),

    ('pure-beeswax-blocks', 'en', 'Pure Beeswax Blocks',
     'Oily wax secreted from bee glands to build honeycombs, used in candles and skin creams. Cast into 100 g blocks and boxed in fours.'),
    ('pure-beeswax-blocks', 'hy', 'Մաքուր մեղրամոմի կտորներ',
     'Յուղոտ մոմ, որը մեղուները արտազատում են գեղձերից՝ խորիսխ կառուցելու համար. օգտագործվում է մոմերի և մաշկի քսուքների մեջ։ Ձուլվում է 100 գ կտորներով, տուփում՝ չորսական։'),
    ('pure-beeswax-blocks', 'ru', 'Чистый пчелиный воск в брусках',
     'Маслянистый воск, который пчёлы выделяют железами для постройки сот; идёт на свечи и кремы для кожи. Отлит брусками по 100 г, в коробке четыре.'),

    ('raw-propolis-tincture', 'en', 'Raw Propolis Tincture',
     'Resinous bee glue collected from plants to seal and protect the hive, known for antimicrobial properties. Bottled in a dropper flask.'),
    ('raw-propolis-tincture', 'hy', 'Հում պրոպոլիսի թուրմ',
     'Բույսերից հավաքված խեժային մեղվասոսինձ, որով մեղուները փակում և պաշտպանում են փեթակը. հայտնի է հակամանրէային հատկություններով։ Շշալցվում է կաթոցիկով սրվակի մեջ։'),
    ('raw-propolis-tincture', 'ru', 'Настойка сырого прополиса',
     'Смолистый пчелиный клей, собранный с растений: им пчёлы запечатывают и защищают улей. Известен антимикробными свойствами. Разлит во флакон с пипеткой.'),

    ('fresh-royal-jelly', 'en', 'Fresh Royal Jelly',
     'Milky fluid fed to queen larvae, used in health supplements and cosmetics. Kept chilled in a small dark jar from the hive to your door.'),
    ('fresh-royal-jelly', 'hy', 'Թարմ մայրական կաթ',
     'Կաթնային հեղուկ, որով կերակրվում են թագուհու թրթուրները. օգտագործվում է սննդային հավելումների և կոսմետիկայի մեջ։ Փեթակից մինչև ձեր դուռը պահվում է սառը՝ մուգ ապակե փոքրիկ բանկայում։'),
    ('fresh-royal-jelly', 'ru', 'Свежее маточное молочко',
     'Молочная масса, которой кормят личинок будущей матки; используется в добавках и косметике. От улья до вашей двери едет охлаждённым, в маленькой тёмной банке.'),

    ('bee-pollen-granules', 'en', 'Bee Pollen Granules',
     'Flower pollen packed by bees, rich in protein and nutrients. Dried gently and sealed in a resealable pouch.'),
    ('bee-pollen-granules', 'hy', 'Մեղվի ծաղկափոշու հատիկներ',
     'Մեղուների հավաքած ծաղկափոշի՝ հարուստ սպիտակուցով և հանքանյութերով։ Մեղմ չորացվում և փակվում է կրկին փակվող տոպրակի մեջ։'),
    ('bee-pollen-granules', 'ru', 'Гранулы пчелиной пыльцы',
     'Цветочная пыльца, собранная пчёлами: много белка и минералов. Бережно высушена и запечатана в пакет с застёжкой.'),

    ('bee-venom-serum', 'en', 'Bee Venom Serum',
     'Secreted defense fluid sometimes used in alternative therapies. Blended into a light serum in a small glass bottle.'),
    ('bee-venom-serum', 'hy', 'Մեղվի թույնով շիճուկ',
     'Պաշտպանական թույն, որը երբեմն կիրառվում է այլընտրանքային բուժման մեջ։ Խառնվում է թեթև շիճուկի մեջ՝ փոքրիկ ապակե սրվակով։'),
    ('bee-venom-serum', 'ru', 'Сыворотка с пчелиным ядом',
     'Защитный яд, который иногда применяют в альтернативной терапии. Смешан в лёгкую сыворотку во флаконе из стекла.')
) AS v(product_slug, locale, name, description)
JOIN products p ON p.slug = v.product_slug
ON CONFLICT (product_id, locale) DO UPDATE
    SET name = EXCLUDED.name, description = EXCLUDED.description;

-- ── "Good for" (many-to-many) ─────────────────────────────────────────────
-- The taxonomy itself is created by migration 000008; this is only which
-- product belongs to which chip. Several products per benefit and several
-- benefits per product — this is exactly the shape a category column cannot
-- express, which is why the join table exists.
--
-- With this mapping the sidebar reads: Energy 3, Immunity 1, Skin 3,
-- Recovery 3, Sweetening 1 — the numbers the facet tests assert.
INSERT INTO product_benefits (product_id, benefit_id)
SELECT p.id, b.id
FROM (VALUES
    ('mountain-wildflower-honey', 'energy'),
    ('mountain-wildflower-honey', 'sweetening'),
    ('pure-beeswax-blocks',       'skin'),
    ('raw-propolis-tincture',     'immunity'),
    ('fresh-royal-jelly',         'energy'),
    ('fresh-royal-jelly',         'skin'),
    ('fresh-royal-jelly',         'recovery'),
    ('bee-pollen-granules',       'energy'),
    ('bee-pollen-granules',       'recovery'),
    ('bee-venom-serum',           'recovery'),
    ('bee-venom-serum',           'skin')
) AS v(product_slug, benefit_slug)
JOIN products p ON p.slug = v.product_slug
JOIN benefits b ON b.slug = v.benefit_slug
-- Nothing to update: the row IS the fact, so a conflict means it is already
-- true. This is the one place DO NOTHING is right rather than lazy.
ON CONFLICT (product_id, benefit_id) DO NOTHING;

-- ── Variants ──────────────────────────────────────────────────────────────
-- Labels are PURE MEASUREMENTS, the E2 decision: a measurement means the
-- same thing in Armenian, Russian and English, so the column stays
-- locale-invariant next to sku and price_minor and needs no fourth
-- translation table. The mock's "500 g jar" loses its noun here and gains it
-- back in the product copy above.
--
-- The cheapest variant of each product is the size the design's card shows,
-- so the "from" price on the card matches the mock. Royal jelly keeps the
-- mock's deliberately NON-LINEAR pricing (25 g $32, 50 g $58, 100 g $105) —
-- twice the jelly is not twice the price, which the per-variant price_minor
-- column has always been able to express.
--
-- RJL-FRS-100 is seeded at zero stock on purpose: sold-out is one of the
-- states the design never draws (§6 exception 2), and a dev database where
-- nothing is ever out of stock is a dev database where that state is never
-- looked at.
INSERT INTO product_variants (product_id, sku, label, stock_qty)
SELECT p.id, v.sku, v.label, v.stock_qty
FROM (VALUES
    ('mountain-wildflower-honey', 'HON-WLD-500', '500 g',     40),
    ('mountain-wildflower-honey', 'HON-WLD-1000', '1 kg',     18),
    ('pure-beeswax-blocks',       'WAX-BLK-4X100', '4 × 100 g', 25),
    ('pure-beeswax-blocks',       'WAX-BLK-10X100', '10 × 100 g', 8),
    ('raw-propolis-tincture',     'PRO-TNC-30',  '30 ml',     30),
    ('raw-propolis-tincture',     'PRO-TNC-100', '100 ml',     9),
    ('fresh-royal-jelly',         'RJL-FRS-25',  '25 g',      14),
    ('fresh-royal-jelly',         'RJL-FRS-50',  '50 g',       6),
    ('fresh-royal-jelly',         'RJL-FRS-100', '100 g',      0),
    ('bee-pollen-granules',       'POL-GRN-250', '250 g',     35),
    ('bee-pollen-granules',       'POL-GRN-500', '500 g',     12),
    ('bee-venom-serum',           'VEN-SRM-15',  '15 ml',     20)
) AS v(product_slug, sku, label, stock_qty)
JOIN products p ON p.slug = v.product_slug
ON CONFLICT (sku) DO UPDATE
    SET label = EXCLUDED.label,
        stock_qty = EXCLUDED.stock_qty;

-- ── E5: one shelf price per market ───────────────────────────────────────
--
-- The price left product_variants in migration 000016 and became a row per
-- (variant, currency). Both columns come from the mock's own table, with the
-- same caveat as the dollar figures: §1.1 confirmed every number in the
-- design is placeholder, so these are shaped like real prices rather than
-- being real ones. What they DO demonstrate is the model — the AMD column is
-- not the USD column times a rate, and the shop is free to price a jar
-- keenly in one market and normally in the other.
INSERT INTO variant_prices (variant_id, currency, price_minor)
SELECT v.id, p.currency, p.price_minor
FROM (VALUES
    ('HON-WLD-500',    'USD',  1400), ('HON-WLD-500',    'AMD',  6700),
    ('HON-WLD-1000',   'USD',  2600), ('HON-WLD-1000',   'AMD', 12400),
    ('WAX-BLK-4X100',  'USD',   900), ('WAX-BLK-4X100',  'AMD',  4300),
    ('WAX-BLK-10X100', 'USD',  2000), ('WAX-BLK-10X100', 'AMD',  9600),
    ('PRO-TNC-30',     'USD',  1900), ('PRO-TNC-30',     'AMD',  9100),
    -- PRO-TNC-100 is deliberately left WITHOUT a dram price: it is the one
    -- variant that exercises the converted fallback, so the shop shows a
    -- computed 22,620 ֏ next to its round-numbered neighbours. Same argument
    -- as RJL-FRS-100's zero stock above — a dev database where every path is
    -- perfectly configured is one where the fallback is never looked at.
    ('PRO-TNC-100',    'USD',  5800),
    ('RJL-FRS-25',     'USD',  3200), ('RJL-FRS-25',     'AMD', 15300),
    ('RJL-FRS-50',     'USD',  5800), ('RJL-FRS-50',     'AMD', 27700),
    ('RJL-FRS-100',    'USD', 10500), ('RJL-FRS-100',    'AMD', 50000),
    ('POL-GRN-250',    'USD',  1600), ('POL-GRN-250',    'AMD',  7600),
    ('POL-GRN-500',    'USD',  2900), ('POL-GRN-500',    'AMD', 13900),
    ('VEN-SRM-15',     'USD',  2800), ('VEN-SRM-15',     'AMD', 13400)
) AS p(sku, currency, price_minor)
JOIN product_variants v ON v.sku = p.sku
ON CONFLICT (variant_id, currency) DO UPDATE
    SET price_minor = EXCLUDED.price_minor;

-- Convergence, not tidiness: if an earlier run (or an admin experimenting in
-- dev) gave the 100 ml tincture a dram price, re-running the seed has to take
-- it away again, or the file stops describing the state it produces.
DELETE FROM variant_prices vp
USING product_variants v
WHERE vp.variant_id = v.id
  AND vp.currency = 'AMD'
  AND v.sku = 'PRO-TNC-100';

-- ══════════════════════════════════════════════════════════════════════════
-- E3: the product page's editorial half
-- ══════════════════════════════════════════════════════════════════════════
--
-- A NOTE ON HOW MUCH OF THIS IS REAL. The design wrote full product-page copy
-- for ONE product (royal jelly); everything below for the other five is
-- plausible placeholder prose, and the family's to replace. So the languages
-- are seeded unevenly, on purpose:
--
--   * notes and highlights — all six products, all three languages. They are
--     short and mechanical enough to be worth translating now.
--   * usage cards — all six in English, plus Armenian and Russian for royal
--     jelly, the product the design actually wrote. The rest deliberately
--     fall back to English, which also makes the per-LIST fallback visible in
--     a running shop rather than only in a test.
--
-- Armenian and Russian remain machine-assisted and flagged for native review.

-- ── Locale-invariant metadata ─────────────────────────────────────────────
UPDATE products p
SET lab_batch = v.lab_batch, is_cold_chain = v.is_cold_chain
FROM (VALUES
    ('mountain-wildflower-honey', 'WH-0626', FALSE),
    ('pure-beeswax-blocks',       'BW-0526', FALSE),
    ('raw-propolis-tincture',     'PR-0426', FALSE),
    -- The one product the design marks "Cold chain", and the reason the
    -- column is a BOOLEAN: E6 charges chilled shipping off this, and a
    -- translated string could not be reasoned about.
    ('fresh-royal-jelly',         'RJ-0626', TRUE),
    ('bee-pollen-granules',       'BP-0626', FALSE),
    ('bee-venom-serum',           'BV-0326', FALSE)
) AS v(slug, lab_batch, is_cold_chain)
WHERE p.slug = v.slug;

-- ── Per-language notes ────────────────────────────────────────────────────
-- An UPDATE, not an INSERT: the translation rows already exist from the E2
-- section above, and these are four more columns on them (migration 000013).
UPDATE product_translations t
SET disclaimer    = v.disclaimer,
    storage_note  = v.storage_note,
    harvest_note  = v.harvest_note,
    shipping_note = v.shipping_note
FROM (VALUES
    ('mountain-wildflower-honey', 'en',
     'A food, not a medicine. Not for infants under one year.',
     'Keep the jar closed at room temperature, away from direct sun. Crystallisation is normal and reverses in a warm water bath.',
     'August 2026, Hives 12–18', 'Ships in 2–4 days'),
    ('mountain-wildflower-honey', 'hy',
     'Սնունդ է, ոչ դեղամիջոց։ Չտալ մեկ տարեկանից փոքր երեխաներին։',
     'Պահել փակ բանկայում՝ սենյակային ջերմաստիճանում, արևից հեռու։ Բյուրեղացումը բնական է և վերանում է տաք ջրի բաղնիքում։',
     '2026 օգոստոս, փեթակներ 12–18', 'Առաքվում է 2–4 օրում'),
    ('mountain-wildflower-honey', 'ru',
     'Продукт питания, а не лекарство. Не давать детям до года.',
     'Держите банку закрытой при комнатной температуре, вдали от солнца. Кристаллизация естественна и уходит на водяной бане.',
     'Август 2026, ульи 12–18', 'Доставка 2–4 дня'),

    ('pure-beeswax-blocks', 'en',
     'For external and craft use. Not intended to be eaten.',
     'Keeps for years in a cool, dry cupboard. No refrigeration needed.',
     'July 2026, Hives 3–9', 'Ships in 2–4 days'),
    ('pure-beeswax-blocks', 'hy',
     'Արտաքին և արհեստագործական օգտագործման համար։ Նախատեսված չէ ուտելու։',
     'Պահպանվում է տարիներ՝ զով, չոր պահարանում։ Սառնարան պետք չէ։',
     '2026 հուլիս, փեթակներ 3–9', 'Առաքվում է 2–4 օրում'),
    ('pure-beeswax-blocks', 'ru',
     'Для наружного и ремесленного применения. Не предназначен в пищу.',
     'Хранится годами в прохладном сухом шкафу. Холодильник не нужен.',
     'Июль 2026, ульи 3–9', 'Доставка 2–4 дня'),

    ('raw-propolis-tincture', 'en',
     'Not a medicine. Avoid if you are allergic to bee products.',
     'Store the bottle upright in a dark place. The tincture keeps its strength for two years.',
     'April 2026, Hives 20–24', 'Ships in 2–4 days'),
    ('raw-propolis-tincture', 'hy',
     'Դեղամիջոց չէ։ Խուսափեք, եթե ալերգիա ունեք մեղվի արտադրանքի նկատմամբ։',
     'Պահել շիշը ուղղահայաց՝ մութ տեղում։ Թուրմը պահպանում է ուժը երկու տարի։',
     '2026 ապրիլ, փեթակներ 20–24', 'Առաքվում է 2–4 օրում'),
    ('raw-propolis-tincture', 'ru',
     'Не лекарство. Избегайте при аллергии на продукты пчеловодства.',
     'Храните флакон вертикально в тёмном месте. Настойка держит силу два года.',
     'Апрель 2026, ульи 20–24', 'Доставка 2–4 дня'),

    ('fresh-royal-jelly', 'en',
     'Not a medicine. Avoid if you are allergic to bee products.',
     'Keep refrigerated at 2–5 °C and closed between doses. Do not freeze — freezing breaks the proteins this is worth eating for.',
     'June 2026, Hive 41', 'Chilled, 2–4 days'),
    ('fresh-royal-jelly', 'hy',
     'Դեղամիջոց չէ։ Խուսափեք, եթե ալերգիա ունեք մեղվի արտադրանքի նկատմամբ։',
     'Պահել սառնարանում՝ 2–5 °C, և փակ պահել դեղաչափերի միջև։ Չսառեցնել — սառեցումը քայքայում է հենց այն սպիտակուցները, որոնց համար այն արժե։',
     '2026 հունիս, փեթակ 41', 'Սառը շղթայով, 2–4 օր'),
    ('fresh-royal-jelly', 'ru',
     'Не лекарство. Избегайте при аллергии на продукты пчеловодства.',
     'Держите в холодильнике при 2–5 °C и закрытым между приёмами. Не замораживать — заморозка разрушает те самые белки, ради которых его берут.',
     'Июнь 2026, улей 41', 'Охлаждённая доставка, 2–4 дня'),

    ('bee-pollen-granules', 'en',
     'A food supplement, not a medicine. Start with a small dose if you have not eaten pollen before.',
     'Reseal the pouch after each use and keep it dry. Refrigeration extends its life but is not required.',
     'May 2026, Hives 5–11', 'Ships in 2–4 days'),
    ('bee-pollen-granules', 'hy',
     'Սննդային հավելում է, ոչ դեղամիջոց։ Սկսեք փոքր չափաբաժնից, եթե նախկինում ծաղկափոշի չեք կերել։',
     'Ամեն օգտագործումից հետո փակեք տոպրակը և պահեք չոր։ Սառնարանը երկարացնում է ժամկետը, բայց պարտադիր չէ։',
     '2026 մայիս, փեթակներ 5–11', 'Առաքվում է 2–4 օրում'),
    ('bee-pollen-granules', 'ru',
     'Пищевая добавка, а не лекарство. Начните с малой дозы, если раньше не ели пыльцу.',
     'Закрывайте пакет после каждого раза и держите сухим. Холодильник продлевает срок, но не обязателен.',
     'Май 2026, ульи 5–11', 'Доставка 2–4 дня'),

    ('bee-venom-serum', 'en',
     'Not a medicine. Patch-test on a small area first, and do not use if you are allergic to bee stings.',
     'Room temperature, cap closed, away from sunlight. Use within six months of opening.',
     'March 2026, Hives 30–33', 'Ships in 2–4 days'),
    ('bee-venom-serum', 'hy',
     'Դեղամիջոց չէ։ Նախ փորձարկեք փոքր հատվածի վրա և մի օգտագործեք, եթե ալերգիա ունեք մեղվի խայթոցի նկատմամբ։',
     'Սենյակային ջերմաստիճան, փակ խցան, արևից հեռու։ Օգտագործել բացելուց հետո վեց ամսվա ընթացքում։',
     '2026 մարտ, փեթակներ 30–33', 'Առաքվում է 2–4 օրում'),
    ('bee-venom-serum', 'ru',
     'Не лекарство. Сначала сделайте тест на небольшом участке и не используйте при аллергии на укусы пчёл.',
     'Комнатная температура, флакон закрыт, вдали от солнца. Использовать в течение шести месяцев после вскрытия.',
     'Март 2026, ульи 30–33', 'Доставка 2–4 дня')
) AS v(product_slug, locale, disclaimer, storage_note, harvest_note, shipping_note)
JOIN products p ON p.slug = v.product_slug
WHERE t.product_id = p.id AND t.locale = v.locale;

-- ── "What it does" bullets ────────────────────────────────────────────────
-- Rows keyed by (product, locale, position) — decision #4. sort_order is
-- stated explicitly here rather than derived, because a seed is a literal
-- description of a desired state, not a form submission.
--
-- DELETE first: the PK includes sort_order, so an upsert would leave a
-- stale fourth bullet behind if a later edit shortens a list. Re-running the
-- seed must CONVERGE, which for a positional collection means replace.
DELETE FROM product_highlights
WHERE product_id IN (SELECT id FROM products WHERE slug IN (
    'mountain-wildflower-honey', 'pure-beeswax-blocks', 'raw-propolis-tincture',
    'fresh-royal-jelly', 'bee-pollen-granules', 'bee-venom-serum'));

INSERT INTO product_highlights (product_id, locale, sort_order, text)
SELECT p.id, v.locale, v.sort_order, v.text
FROM (VALUES
    ('mountain-wildflower-honey', 'en', 0, 'Steady natural energy from fruit sugars, not a caffeine spike'),
    ('mountain-wildflower-honey', 'en', 1, 'Unfiltered, so the pollen and enzymes are still in the jar'),
    ('mountain-wildflower-honey', 'en', 2, 'One meadow, one season — never blended across harvests'),
    ('mountain-wildflower-honey', 'hy', 0, 'Կայուն բնական էներգիա մրգային շաքարներից, ոչ թե կոֆեինային ցատկ'),
    ('mountain-wildflower-honey', 'hy', 1, 'Չզտված՝ ծաղկափոշին և ֆերմենտները մնում են բանկայում'),
    ('mountain-wildflower-honey', 'hy', 2, 'Մեկ մարգագետին, մեկ սեզոն — երբեք չի խառնվում բերքերի միջև'),
    ('mountain-wildflower-honey', 'ru', 0, 'Ровная природная энергия от фруктовых сахаров, без кофеинового скачка'),
    ('mountain-wildflower-honey', 'ru', 1, 'Нефильтрованный — пыльца и ферменты остаются в банке'),
    ('mountain-wildflower-honey', 'ru', 2, 'Один луг, один сезон — никогда не смешиваем урожаи'),

    ('pure-beeswax-blocks', 'en', 0, 'Melts clean for balms, salves and creams'),
    ('pure-beeswax-blocks', 'en', 1, 'Burns slowly and without soot in poured candles'),
    ('pure-beeswax-blocks', 'en', 2, 'Nothing added — no paraffin, no bleaching, no scent'),
    ('pure-beeswax-blocks', 'hy', 0, 'Մաքուր հալվում է քսուքների, բալզամների և կրեմների համար'),
    ('pure-beeswax-blocks', 'hy', 1, 'Այրվում է դանդաղ և առանց մրի ձուլված մոմերի մեջ'),
    ('pure-beeswax-blocks', 'hy', 2, 'Ոչինչ ավելացված չէ — ոչ պարաֆին, ոչ սպիտակեցում, ոչ բույր'),
    ('pure-beeswax-blocks', 'ru', 0, 'Чисто плавится для бальзамов, мазей и кремов'),
    ('pure-beeswax-blocks', 'ru', 1, 'Горит медленно и без копоти в литых свечах'),
    ('pure-beeswax-blocks', 'ru', 2, 'Ничего не добавлено — ни парафина, ни отбеливания, ни отдушек'),

    ('raw-propolis-tincture', 'en', 0, 'Known for antimicrobial and antifungal activity'),
    ('raw-propolis-tincture', 'en', 1, 'Traditionally taken at the first sign of a sore throat'),
    ('raw-propolis-tincture', 'en', 2, 'Collected by hand from the hive, never scraped from frames'),
    ('raw-propolis-tincture', 'hy', 0, 'Հայտնի է հակամանրէային և հակասնկային ազդեցությամբ'),
    ('raw-propolis-tincture', 'hy', 1, 'Ավանդաբար ընդունվում է կոկորդի ցավի առաջին նշանից'),
    ('raw-propolis-tincture', 'hy', 2, 'Հավաքվում է ձեռքով փեթակից, երբեք չի քերվում շրջանակներից'),
    ('raw-propolis-tincture', 'ru', 0, 'Известен противомикробным и противогрибковым действием'),
    ('raw-propolis-tincture', 'ru', 1, 'Традиционно принимают при первых признаках боли в горле'),
    ('raw-propolis-tincture', 'ru', 2, 'Собран вручную из улья, а не соскоблен с рамок'),

    ('fresh-royal-jelly', 'en', 0, 'Supports energy and stamina through the season change'),
    ('fresh-royal-jelly', 'en', 1, 'Used in cosmetics for skin elasticity and repair'),
    ('fresh-royal-jelly', 'en', 2, 'Rich in B vitamins, amino acids and proteins'),
    ('fresh-royal-jelly', 'hy', 0, 'Աջակցում է էներգիային և տոկունությանը եղանակների փոփոխության ժամանակ'),
    ('fresh-royal-jelly', 'hy', 1, 'Օգտագործվում է կոսմետիկայում՝ մաշկի առաձգականության և վերականգնման համար'),
    ('fresh-royal-jelly', 'hy', 2, 'Հարուստ է B խմբի վիտամիններով, ամինաթթուներով և սպիտակուցներով'),
    ('fresh-royal-jelly', 'ru', 0, 'Поддерживает энергию и выносливость при смене сезона'),
    ('fresh-royal-jelly', 'ru', 1, 'Используется в косметике для упругости и восстановления кожи'),
    ('fresh-royal-jelly', 'ru', 2, 'Богато витаминами группы B, аминокислотами и белками'),

    ('bee-pollen-granules', 'en', 0, 'Roughly a quarter protein by weight, with all the essential amino acids'),
    ('bee-pollen-granules', 'en', 1, 'A slow, even lift rather than a sugar rush'),
    ('bee-pollen-granules', 'en', 2, 'Dried below hive temperature, so the enzymes survive'),
    ('bee-pollen-granules', 'hy', 0, 'Կշռի մոտ քառորդը սպիտակուց է՝ բոլոր անհրաժեշտ ամինաթթուներով'),
    ('bee-pollen-granules', 'hy', 1, 'Դանդաղ, հավասար վերելք, ոչ թե շաքարային ցատկ'),
    ('bee-pollen-granules', 'hy', 2, 'Չորացվում է փեթակի ջերմաստիճանից ցածր՝ ֆերմենտները պահպանվում են'),
    ('bee-pollen-granules', 'ru', 0, 'Около четверти веса — белок, со всеми незаменимыми аминокислотами'),
    ('bee-pollen-granules', 'ru', 1, 'Медленный ровный подъём, а не сахарный скачок'),
    ('bee-pollen-granules', 'ru', 2, 'Сушится ниже температуры улья, поэтому ферменты сохраняются'),

    ('bee-venom-serum', 'en', 0, 'Used in apitherapy for stiff joints and tired muscles'),
    ('bee-venom-serum', 'en', 1, 'Blended at a low concentration into a light, fast-absorbing base'),
    ('bee-venom-serum', 'en', 2, 'Venom collected without harming the bees'),
    ('bee-venom-serum', 'hy', 0, 'Օգտագործվում է ապիթերապիայում՝ կարկամած հոդերի և հոգնած մկանների համար'),
    ('bee-venom-serum', 'hy', 1, 'Խառնված է ցածր խտությամբ՝ թեթև, արագ ներծծվող հիմքի մեջ'),
    ('bee-venom-serum', 'hy', 2, 'Թույնը հավաքվում է առանց մեղուներին վնասելու'),
    ('bee-venom-serum', 'ru', 0, 'Применяется в апитерапии при скованных суставах и усталых мышцах'),
    ('bee-venom-serum', 'ru', 1, 'Смешан в низкой концентрации с лёгкой, быстро впитывающейся основой'),
    ('bee-venom-serum', 'ru', 2, 'Яд собирается без вреда для пчёл')
) AS v(product_slug, locale, sort_order, text)
JOIN products p ON p.slug = v.product_slug;

-- ── Usage cards (Morning / Course / Pairs with) ───────────────────────────
-- English for all six; Armenian and Russian for royal jelly only — see the
-- note at the top of this section. The other five fall back as a whole list,
-- which is the behaviour store.attachUsageCards implements.
DELETE FROM product_usage_cards
WHERE product_id IN (SELECT id FROM products WHERE slug IN (
    'mountain-wildflower-honey', 'pure-beeswax-blocks', 'raw-propolis-tincture',
    'fresh-royal-jelly', 'bee-pollen-granules', 'bee-venom-serum'));

INSERT INTO product_usage_cards (product_id, locale, sort_order, kicker, title, body)
SELECT p.id, v.locale, v.sort_order, v.kicker, v.title, v.body
FROM (VALUES
    ('mountain-wildflower-honey', 'en', 0, 'Morning', 'A spoon, plain', 'On bread or straight from the spoon. Stirring it into boiling tea undoes most of what it is here for.'),
    ('mountain-wildflower-honey', 'en', 1, 'Kitchen', 'Off the heat', 'Add at the end of cooking, not the start — above 40 °C the enzymes go and you are left with sweetness alone.'),
    ('mountain-wildflower-honey', 'en', 2, 'Pairs with', 'Pollen and jelly', 'A spoon of honey is the easiest way to take the sharper things on this shelf.'),

    ('pure-beeswax-blocks', 'en', 0, 'Balms', 'One part wax', 'Three parts oil to one part wax by weight gives a salve that holds its shape in a tin without going hard.'),
    ('pure-beeswax-blocks', 'en', 1, 'Candles', 'Melt low and slow', 'A water bath, never direct heat. Beeswax scorches, and scorched wax smells like nothing you want in a room.'),
    ('pure-beeswax-blocks', 'en', 2, 'Pairs with', 'Propolis', 'A few drops of tincture in a balm base is the classic winter hand salve.'),

    ('raw-propolis-tincture', 'en', 0, 'Daily', 'Ten drops in water', 'The resin will cloud the glass — that is the propolis coming out of the alcohol, not a fault.'),
    ('raw-propolis-tincture', 'en', 1, 'Course', 'Two weeks on', 'Then a week off. Most people run it through the changeable weeks of autumn.'),
    ('raw-propolis-tincture', 'en', 2, 'Pairs with', 'Honey', 'A spoon of honey after the drops takes the sting out of the taste.'),

    ('fresh-royal-jelly', 'en', 0, 'Morning', 'A grain of rice', 'Under the tongue before breakfast, on an empty stomach. Let it dissolve rather than swallowing.'),
    ('fresh-royal-jelly', 'en', 1, 'Course', 'Three weeks on', 'Then a week off. Most people run a course at the turn of autumn and again in early spring.'),
    ('fresh-royal-jelly', 'en', 2, 'Pairs with', 'Honey and pollen', 'Stir the day''s dose into a spoon of wildflower honey if the taste is too sharp on its own.'),
    ('fresh-royal-jelly', 'hy', 0, 'Առավոտյան', 'Բրնձի հատիկի չափ', 'Լեզվի տակ՝ նախաճաշից առաջ, դատարկ ստամոքսին։ Թողեք, որ լուծվի, մի կուլ տվեք։'),
    ('fresh-royal-jelly', 'hy', 1, 'Կուրս', 'Երեք շաբաթ', 'Ապա մեկ շաբաթ ընդմիջում։ Շատերն անցնում են կուրս աշնան սկզբին և կրկին վաղ գարնանը։'),
    ('fresh-royal-jelly', 'hy', 2, 'Զուգակցվում է', 'Մեղրի և ծաղկափոշու հետ', 'Խառնեք օրվա չափաբաժինը մի գդալ վայրի ծաղիկների մեղրի մեջ, եթե համը շատ սուր է։'),
    ('fresh-royal-jelly', 'ru', 0, 'Утром', 'С рисовое зерно', 'Под язык до завтрака, натощак. Дайте раствориться, не глотайте.'),
    ('fresh-royal-jelly', 'ru', 1, 'Курс', 'Три недели', 'Затем неделя перерыва. Обычно курс проходят на переломе осени и ещё раз ранней весной.'),
    ('fresh-royal-jelly', 'ru', 2, 'Сочетается с', 'Мёдом и пыльцой', 'Размешайте дневную дозу в ложке цветочного мёда, если вкус кажется слишком резким.'),

    ('bee-pollen-granules', 'en', 0, 'Start', 'A few granules', 'Build up over a week. Pollen is a common allergen and a small first dose is the sensible way in.'),
    ('bee-pollen-granules', 'en', 1, 'Daily', 'A teaspoon', 'In yoghurt, on porridge, or chewed on their own if you like the taste of a meadow.'),
    ('bee-pollen-granules', 'en', 2, 'Pairs with', 'Honey', 'Honey and pollen together are the oldest breakfast on this shelf.'),

    ('bee-venom-serum', 'en', 0, 'First', 'Patch-test', 'Inside the forearm, and wait a day. Bee venom is exactly as serious as it sounds if you react to stings.'),
    ('bee-venom-serum', 'en', 1, 'Use', 'A thin layer', 'Massage into the joint or muscle until it disappears. A warm tingle is expected; burning is not.'),
    ('bee-venom-serum', 'en', 2, 'Pairs with', 'Beeswax balm', 'A wax balm over the top holds the serum against the skin for longer.')
) AS v(product_slug, locale, sort_order, kicker, title, body)
JOIN products p ON p.slug = v.product_slug;

-- ── Curated "Often taken together" ────────────────────────────────────────
-- Only royal jelly is curated, on purpose: it exercises the CURATED path
-- while the other five demonstrate the computed fallback (shared benefits,
-- then popularity). Two behaviours visible in one seeded shop.
--
-- The pairings come from that product's own usage cards, which is what the
-- panel is really claiming: "people take these together".
DELETE FROM product_related
WHERE product_id IN (SELECT id FROM products WHERE slug = 'fresh-royal-jelly');

INSERT INTO product_related (product_id, related_id, sort_order)
SELECT src.id, dst.id, v.sort_order
FROM (VALUES
    ('fresh-royal-jelly', 'mountain-wildflower-honey', 0),
    ('fresh-royal-jelly', 'bee-pollen-granules',       1),
    ('fresh-royal-jelly', 'raw-propolis-tincture',     2)
) AS v(product_slug, related_slug, sort_order)
JOIN products src ON src.slug = v.product_slug
JOIN products dst ON dst.slug = v.related_slug
ON CONFLICT (product_id, related_id) DO UPDATE SET sort_order = EXCLUDED.sort_order;

-- ══════════════════════════════════════════════════════════════════════════
-- E4: reviews
-- ══════════════════════════════════════════════════════════════════════════
--
-- Seeded reviews need seeded REVIEWERS, and a reviewer needs a delivered
-- order — the verified-purchase rule is enforced in the store, not just in
-- the UI, so there is no shortcut here that the application would accept.
-- This section therefore builds the whole chain: users → orders (delivered)
-- → order_items → reviews.
--
-- The passwords are a fixed bcrypt hash of "seed-password-123". These are
-- sample customers for a dev database, not accounts anyone should be able to
-- sign in to on a real deployment — which is another reason seed.sql is not
-- run by the compose stacks.

INSERT INTO users (email, password_hash, role) VALUES
    ('anahit@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'customer'),
    ('vahe@example.com',   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'customer'),
    ('mariam@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'customer'),
    ('sergey@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'customer')
ON CONFLICT (email) DO NOTHING;

-- One delivered order per reviewer, containing the cheapest variant of every
-- product they review. `delivered` is the point: the store checks for that
-- exact status, so an order in any other state grants no standing.
INSERT INTO orders (user_id, status, total_minor, currency)
SELECT u.id, 'delivered', 0, 'USD'
FROM users u
WHERE u.email IN ('anahit@example.com', 'vahe@example.com',
                  'mariam@example.com', 'sergey@example.com')
  AND NOT EXISTS (
      SELECT 1 FROM orders o WHERE o.user_id = u.id AND o.status = 'delivered'
  );

INSERT INTO order_items (order_id, variant_id, name_snapshot, label_snapshot,
                         price_minor_snapshot, qty)
SELECT o.id, v.id, p.name, v.label, v.price_minor, 1
FROM orders o
JOIN users u ON u.id = o.user_id
CROSS JOIN products p
-- "Cheapest variant" is now a question with a market attached, and these
-- orders are stamped USD, so the effective price view is asked in USD.
JOIN LATERAL (
    SELECT pv.id, pv.label, ep.price_minor
    FROM product_variants pv
    JOIN variant_effective_prices ep
      ON ep.variant_id = pv.id AND ep.currency = o.currency
    WHERE pv.product_id = p.id
    ORDER BY ep.price_minor
    LIMIT 1
) v ON TRUE
WHERE o.status = 'delivered'
  AND u.email IN ('anahit@example.com', 'vahe@example.com',
                  'mariam@example.com', 'sergey@example.com')
  AND p.is_active
  AND NOT EXISTS (
      SELECT 1 FROM order_items oi WHERE oi.order_id = o.id AND oi.variant_id = v.id
  );

-- The reviews themselves. A spread of statuses on purpose: `published` rows
-- feed the storefront and the aggregate, and the two `pending` ones give the
-- admin moderation queue something to actually moderate.
INSERT INTO reviews (product_id, user_id, rating, title, body, status)
SELECT p.id, u.id, v.rating, v.title, v.body, v.status
FROM (VALUES
    ('mountain-wildflower-honey', 'anahit@example.com', 5, 'Tastes like the meadow',
     'Thick, unfiltered, and it crystallised in the cupboard exactly as they said it would. The warm water bath brings it straight back.', 'published'),
    ('mountain-wildflower-honey', 'vahe@example.com', 5, 'We go through a jar a month',
     'Bought the 1 kg after finishing the small one in two weeks. Nothing else tastes like it.', 'published'),
    ('mountain-wildflower-honey', 'mariam@example.com', 4, 'Lovely, if pricier than the shop',
     'Worth it for the flavour. Four stars only because I wish the 1 kg came in glass.', 'published'),

    ('fresh-royal-jelly', 'anahit@example.com', 5, 'Arrived properly cold',
     'The cold chain is real — it came in a chilled box and went straight into the fridge. Sharp taste, exactly as described.', 'published'),
    ('fresh-royal-jelly', 'sergey@example.com', 4, 'Doing the three-week course',
     'Too early to say much about the effect, but the quality is obvious and the instructions on the page were clear.', 'published'),

    ('raw-propolis-tincture', 'vahe@example.com', 5, 'Kept a sore throat away',
     'Ten drops in water at the first scratchy morning. Tastes medicinal, which I take as a good sign.', 'published'),
    ('raw-propolis-tincture', 'mariam@example.com', 3, 'Works, but the taste is brutal',
     'No complaints about the product itself. Follow their advice and chase it with honey.', 'published'),

    ('bee-pollen-granules', 'sergey@example.com', 5, 'On the morning yoghurt',
     'Started with a few granules as suggested. No reaction, and a much steadier morning than coffee gave me.', 'published'),

    ('pure-beeswax-blocks', 'mariam@example.com', 5, 'Perfect for balms',
     'Three parts oil to one part wax, exactly as their card says. Clean melt, no smell, no soot in the candles either.', 'published'),

    -- Awaiting moderation: the queue needs rows, and these also prove that
    -- pending reviews do NOT move the public average.
    ('bee-venom-serum', 'anahit@example.com', 4, 'Careful with the patch test',
     'Did the forearm test for a full day first, as they say to. Warm tingle, no reaction. Early days.', 'pending'),
    ('bee-pollen-granules', 'vahe@example.com', 2, 'Not for me',
     'No fault of theirs — I just could not get used to the taste.', 'pending')
) AS v(product_slug, email, rating, title, body, status)
JOIN products p ON p.slug = v.product_slug
JOIN users u ON u.email = v.email
ON CONFLICT (product_id, user_id) DO UPDATE
    SET rating = EXCLUDED.rating,
        title  = EXCLUDED.title,
        body   = EXCLUDED.body,
        status = EXCLUDED.status;

-- The aggregate, recomputed the same way store.recomputeRating does it.
--
-- The seed writes reviews with plain INSERTs, bypassing the application
-- entirely — so it also has to do the application's job of keeping the
-- denormalized columns honest. That is the standing cost of denormalizing,
-- stated in SQL: every writer must maintain it, and a writer that forgets is
-- a silent bug.
UPDATE products p
SET rating_avg   = COALESCE(r.avg_rating, 0),
    rating_count = COALESCE(r.n, 0)
FROM (
    SELECT product_id,
           avg(rating)::numeric(3,2) AS avg_rating,
           count(*) AS n
    FROM reviews
    WHERE status = 'published'
    GROUP BY product_id
) r
WHERE p.id = r.product_id;
