-- Accounts: wishlist, password reset, OAuth identities, named addresses
-- (docs/PLAN_ERA_2.md, phase E8).

-- ── Wishlist ──────────────────────────────────────────────────────────────
--
-- The hearts on every screen. A wishlist entry is a (person, product) fact
-- with no quantity, no variant and no order — which is why it is its own
-- table rather than a flag on cart_items: a cart line answers "what am I
-- buying", a wishlist row answers "what am I keeping an eye on", and the
-- design's "Save for later" is a MOVE between the two.
--
-- The composite PK is the whole identity — hearting twice is the same fact
-- stated twice, so writes are ON CONFLICT DO NOTHING upserts (idempotent,
-- like the cart's set-semantics). Login required, consistent with decision
-- #9 on carts; anonymous wishlists stay in the backlog.
CREATE TABLE wishlist_items (
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    product_id BIGINT      NOT NULL REFERENCES products (id) ON DELETE CASCADE,
    added_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, product_id)
);

-- ── Password reset tokens ─────────────────────────────────────────────────
--
-- The sessions pattern (decision #8), third use: the email carries a raw
-- 256-bit token, the table stores only its SHA-256 — someone who reads this
-- table (a leaked backup, a curious admin) cannot reset anyone's password
-- with what they see. Two columns sessions do not need:
--
--   expires_at  a reset link is a temporary password in an inbox, and
--               inboxes get compromised LATER — a short fuse bounds the
--               damage window. Set by the application (~1 hour).
--   used_at     single use, recorded rather than deleted: "this token was
--               spent at 14:02" is evidence worth keeping if an account is
--               ever disputed, and a NULL check is as cheap as a row check.
CREATE TABLE password_reset_tokens (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id      BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_sha256 TEXT        NOT NULL UNIQUE,
    expires_at   TIMESTAMPTZ NOT NULL,
    used_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Requesting a new link invalidates the old ones (UPDATE … WHERE user_id
-- AND used_at IS NULL); this index is that statement's path.
CREATE INDEX idx_password_reset_tokens_user ON password_reset_tokens (user_id);

-- ── OAuth identities ──────────────────────────────────────────────────────
--
-- "Continue with Google" (decision #5: Google real, Apple a stub on the
-- design). A separate table rather than columns on users because identity
-- and account are different facts: one account may later hold a Google AND
-- an Apple identity, and an account created by OAuth may later gain a
-- password — nothing about either changes the users row.
--
--   provider  the issuer ("google"; "apple" the day the family pays for it)
--   subject   the provider's PERMANENT id for the person ("sub" in OIDC) —
--             never the email, which providers let people change. The email
--             is used ONCE, at first sign-in, to link to an existing
--             account; after that the subject is the identity.
--
-- UNIQUE (provider, subject) is the whole security model of the table: one
-- Google account maps to exactly one shop account, enforced by storage.
CREATE TABLE oauth_identities (
    id         BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider   TEXT        NOT NULL,
    subject    TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (provider, subject)
);

-- An OAuth-created user has no password. The column stays NOT NULL and the
-- value is '' — bcrypt refuses to match ANY input against an empty hash, so
-- password login for such an account fails closed with the same "invalid
-- credentials" as a wrong password, and no schema change ripples through
-- every existing query. Forgot-password later SETS a real hash, which is
-- how such an account gains a password.

-- ── Named addresses ───────────────────────────────────────────────────────
--
-- E6 scoped the book to one default row per user; E8's account page manages
-- several ("Home", "Office", "Mum's"). The one-default-per-user partial
-- unique index from 000017 already permits any number of non-default rows —
-- the only thing missing is a human name for each.
ALTER TABLE addresses
    ADD COLUMN label TEXT NOT NULL DEFAULT '';
