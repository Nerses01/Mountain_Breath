-- F2 (decision #97): account deletion, the privacy page's promise made
-- executable — "Orders we must keep for bookkeeping as the law requires;
-- everything else goes."
--
-- The one schema change deletion needs: an order must be able to OUTLIVE
-- its customer. NULL user_id = "the account that placed this was deleted";
-- every financial fact (items, prices, address snapshot, promo text) is
-- already frozen ON the order, so nothing else is lost with the link.
--
-- The FK deliberately KEEPS its ON DELETE RESTRICT: DeleteAccount detaches
-- orders explicitly inside its transaction, and any OTHER path that tries
-- to delete a user without having decided what happens to their orders
-- still hits the constraint. Same role as reviews' RESTRICT — the schema
-- forces every deleter to make the decision, it does not make it for them.
ALTER TABLE orders
    ALTER COLUMN user_id DROP NOT NULL;
