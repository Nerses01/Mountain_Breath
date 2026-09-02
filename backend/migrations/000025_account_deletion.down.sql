-- Re-imposing NOT NULL requires removing its violators first: orders
-- detached by account deletion. Deleting them here is acceptable ONLY
-- because down migrations run in development — these rows are exactly the
-- bookkeeping the up migration exists to preserve, and a production
-- rollback would need a tombstone-user backfill instead.
-- (order_items and order_status_events cascade; promo_redemptions cannot
-- reference these orders — the account deletion that orphaned them also
-- cascaded its redemption rows away.)
DELETE FROM orders WHERE user_id IS NULL;

ALTER TABLE orders
    ALTER COLUMN user_id SET NOT NULL;
