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
INSERT INTO product_variants (product_id, sku, label, price_minor, stock_qty)
SELECT p.id, v.sku, v.label, v.price_minor, v.stock_qty
FROM (VALUES
    ('mountain-wildflower-honey', 'HON-WLD-500', '500 g',     1400, 40),
    ('mountain-wildflower-honey', 'HON-WLD-1000', '1 kg',     2600, 18),
    ('pure-beeswax-blocks',       'WAX-BLK-4X100', '4 × 100 g', 900, 25),
    ('pure-beeswax-blocks',       'WAX-BLK-10X100', '10 × 100 g', 2000, 8),
    ('raw-propolis-tincture',     'PRO-TNC-30',  '30 ml',     1900, 30),
    ('raw-propolis-tincture',     'PRO-TNC-100', '100 ml',    5800, 9),
    ('fresh-royal-jelly',         'RJL-FRS-25',  '25 g',      3200, 14),
    ('fresh-royal-jelly',         'RJL-FRS-50',  '50 g',      5800, 6),
    ('fresh-royal-jelly',         'RJL-FRS-100', '100 g',    10500, 0),
    ('bee-pollen-granules',       'POL-GRN-250', '250 g',     1600, 35),
    ('bee-pollen-granules',       'POL-GRN-500', '500 g',     2900, 12),
    ('bee-venom-serum',           'VEN-SRM-15',  '15 ml',     2800, 20)
) AS v(product_slug, sku, label, price_minor, stock_qty)
JOIN products p ON p.slug = v.product_slug
ON CONFLICT (sku) DO UPDATE
    SET label = EXCLUDED.label,
        price_minor = EXCLUDED.price_minor,
        stock_qty = EXCLUDED.stock_qty;
