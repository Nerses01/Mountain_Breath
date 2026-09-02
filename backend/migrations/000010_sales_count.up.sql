-- The popularity signal behind the Shop page's default sort, "Most loved"
-- (docs/PLAN_ERA_2.md, phase E2).
--
-- WHY A DENORMALIZED COUNTER instead of aggregating order_items on every
-- list query:
--
--   SELECT p.*, coalesce(sum(oi.qty), 0) AS sold
--   FROM products p
--   LEFT JOIN product_variants v ON v.product_id = p.id
--   LEFT JOIN order_items oi     ON oi.variant_id = v.id
--   GROUP BY p.id
--   ORDER BY sold DESC
--
-- That is correct and needs no column — but it reads EVERY order line ever
-- written to sort six products, and it grows without bound: the shop's
-- busiest year is also its slowest catalog page. It also collides with
-- pagination, since the GROUP BY has to happen before LIMIT can be applied.
--
-- The counter trades storage correctness for read speed. The cost is real
-- and worth naming: the number is now derived data that can drift from the
-- rows it summarises, so every write path that creates or cancels an order
-- has to maintain it, and a bug there is silent (a wrong sort order, not an
-- error). That is an acceptable trade HERE because there is exactly one such
-- path — store.CreateOrder already opens the transaction, so the increment
-- costs one more UPDATE inside a lock we are already holding, and it is
-- atomic with the order it counts. A read-heavy, write-rare counter is the
-- textbook case for denormalizing.
--
-- Deliberately NOT decremented on cancellation: "most loved" is a measure of
-- interest over time, and an order that reached checkout expressed interest
-- whatever happened afterwards. Recorded here so the omission reads as a
-- decision rather than a missing UPDATE in orders.go.
ALTER TABLE products
    ADD COLUMN sales_count INT NOT NULL DEFAULT 0 CHECK (sales_count >= 0);

-- Backfill from the rows the counter summarises, so the column is correct
-- for orders placed before it existed rather than starting every product at
-- zero. This is the aggregate query above, run ONCE.
--
-- The subquery is correlated on v.product_id and returns one row, so it can
-- sit directly in SET. coalesce turns "no orders at all" (NULL from an empty
-- sum) into 0, which the NOT NULL constraint requires.
UPDATE products p
SET sales_count = coalesce((
    SELECT sum(oi.qty)
    FROM order_items oi
    JOIN product_variants v ON v.id = oi.variant_id
    WHERE v.product_id = p.id
), 0);

-- Sorting by popularity reads this column on every Shop page load.
-- DESC matches the ORDER BY direction: a btree index can be scanned
-- backwards, so the direction is not strictly required, but declaring it
-- keeps the index and the query obviously in agreement.
CREATE INDEX idx_products_sales_count ON products (sales_count DESC);
