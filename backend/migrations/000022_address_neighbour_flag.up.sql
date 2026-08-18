-- A4 (canvas 09, decision log #88): "leave with the neighbour" becomes a
-- fact about the ADDRESS — the canvas draws it on the address card, because
-- it describes the place ("my neighbour at this address accepts parcels"),
-- not one delivery.
--
-- The ORDER keeps its own leave_with_neighbour column untouched: this flag
-- PREFILLS the checkout checkbox, the order snapshots what was actually
-- chosen for that delivery — the same suggestion-vs-record split as the
-- address snapshot itself (migration 000017).
ALTER TABLE addresses
    ADD COLUMN leave_with_neighbour BOOLEAN NOT NULL DEFAULT FALSE;
