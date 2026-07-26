-- Development seed data. Idempotent: ON CONFLICT DO NOTHING makes it safe
-- to run repeatedly. Run with:
--   Get-Content backend\seed\seed.sql | docker exec -i mb-postgres psql -U mb -d mountain_breath

INSERT INTO categories (slug, name, sort_order) VALUES
    ('herbal-tea', 'Herbal Tea', 1),
    ('coffee', 'Coffee', 2),
    ('honey', 'Honey & Sweets', 3)
ON CONFLICT (slug) DO NOTHING;

-- Insert products by joining a VALUES table to categories, so we never
-- hardcode generated category ids.
INSERT INTO products (category_id, slug, name, description)
SELECT c.id, v.slug, v.name, v.description
FROM (VALUES
    ('herbal-tea', 'mountain-herbal-tea', 'Mountain Herbal Tea',
     'Wild-picked herbal blend from high mountain meadows.'),
    ('herbal-tea', 'wild-thyme-tea', 'Wild Thyme Tea',
     'Fragrant wild thyme, hand-collected and sun-dried.'),
    ('coffee', 'armenian-coffee', 'Armenian Ground Coffee',
     'Finely ground coffee, roasted for the traditional cezve.'),
    ('honey', 'wildflower-honey', 'Wildflower Honey',
     'Raw honey from alpine wildflower fields.')
) AS v(cat_slug, slug, name, description)
JOIN categories c ON c.slug = v.cat_slug
ON CONFLICT (slug) DO NOTHING;

INSERT INTO product_variants (product_id, sku, label, price_minor, stock_qty)
SELECT p.id, v.sku, v.label, v.price_minor, v.stock_qty
FROM (VALUES
    ('mountain-herbal-tea', 'TEA-MNT-100', '100 g',  180000, 25),
    ('mountain-herbal-tea', 'TEA-MNT-250', '250 g',  400000, 10),
    ('wild-thyme-tea',      'TEA-THM-050', '50 g',   120000, 40),
    ('wild-thyme-tea',      'TEA-THM-100', '100 g',  220000, 18),
    ('armenian-coffee',     'COF-ARM-250', '250 g',  350000, 30),
    ('armenian-coffee',     'COF-ARM-500', '500 g',  650000, 12),
    ('wildflower-honey',    'HON-WLD-350', '350 g',  520000, 15),
    ('wildflower-honey',    'HON-WLD-700', '700 g',  950000,  6)
) AS v(product_slug, sku, label, price_minor, stock_qty)
JOIN products p ON p.slug = v.product_slug
ON CONFLICT (sku) DO NOTHING;
