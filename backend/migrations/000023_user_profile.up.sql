-- A5 (canvas 10): the profile fields the account section finally draws —
-- a name for the rail and header (E8 accounts only ever knew an email),
-- a phone for courier contact — plus the ONE notification preference that
-- has a sender to honour it (decision log #87): order-update emails,
-- which Era III F2's status-change mailer will read before sending.
--
-- Empty string, not NULL, for the text fields: every existing consumer
-- then handles exactly one kind of absence (the '' the badge/API fields
-- already taught the codebase), and rendering falls back to the email
-- without a null branch.
--
-- The other canvas toggles (wishlist alerts, SMS) get NO columns here:
-- their columns arrive with their senders, so a column can never be a
-- promise nothing reads.
ALTER TABLE users
    ADD COLUMN full_name TEXT NOT NULL DEFAULT '',
    ADD COLUMN phone     TEXT NOT NULL DEFAULT '',
    ADD COLUMN notify_order_updates BOOLEAN NOT NULL DEFAULT TRUE;
