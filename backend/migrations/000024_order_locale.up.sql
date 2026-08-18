-- F2: the language the checkout happened in, snapshotted onto the order —
-- the same philosophy as its prices and product names. A status-change
-- email is sent days later, triggered by the ADMIN's request, whose
-- negotiated language is the admin's; the customer's language is a fact
-- about the order, so the order records it.
--
-- DEFAULT 'en' doubles as the backfill: every pre-existing order gets
-- English, which is honestly what its mails were. The CHECK hardcodes the
-- set like every *_translations table before it (migration 000007's
-- reasoning: the list of shop languages changes by migration, not by data).
ALTER TABLE orders
    ADD COLUMN locale TEXT NOT NULL DEFAULT 'en' CHECK (locale IN ('en', 'hy', 'ru'));
