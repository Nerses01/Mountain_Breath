-- A2 (account canvas 07, decision log #85): the order tracker dates each
-- step, so status transitions become RECORDED FACTS instead of destructive
-- updates. `orders.status` stays the single source of the CURRENT state
-- (every existing query keeps working); this table is the history — an
-- append-only audit log, one row per transition, written in the same
-- transaction as the UPDATE it describes.
--
-- Append-only is the point: an UPDATE loses the previous value forever,
-- an INSERT beside it loses nothing. This is the cheapest possible event
-- sourcing — the current-state column is kept (reading "what is it now?"
-- must not require folding history), only the timeline is added.
CREATE TABLE order_status_events (
    id         BIGSERIAL PRIMARY KEY,
    order_id   BIGINT      NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    -- The same CHECK list as orders.status (000004): the history may only
    -- contain states the machine knows.
    status     TEXT        NOT NULL
               CHECK (status IN ('pending', 'confirmed', 'shipped', 'delivered', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The one read path: "the timeline of order N, oldest first". The FK alone
-- creates no index (unlike some engines, Postgres never auto-indexes FKs).
CREATE INDEX idx_order_status_events_order
    ON order_status_events (order_id, created_at);

-- Backfill: every existing order gets its `pending` event, dated by the one
-- transition timestamp the schema ever recorded — creation. Later steps of
-- old orders are honestly unknown (their trackers will show position
-- without dates); inventing timestamps for them would be fiction in an
-- audit table.
INSERT INTO order_status_events (order_id, status, created_at)
SELECT id, 'pending', created_at FROM orders;
