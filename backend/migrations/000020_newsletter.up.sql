-- The newsletter, with double opt-in (docs/PLAN_ERA_2.md, phase E9).
--
-- Double opt-in means typing an address into the footer form proves
-- nothing: anyone can type anyone's address. The row is created UNCONFIRMED
-- and a link goes to the address itself — only the person who can read
-- that inbox can flip confirmed_at, which is why it is the legal default
-- (GDPR-adjacent consent) and the anti-spite default (you cannot sign your
-- neighbour up for mail they never asked for).
--
-- The token is the sessions pattern's FOURTH use (sessions, reset tokens,
-- and E8's oauth state before it): raw 256-bit value in the email, SHA-256
-- here. But with a deliberate twist the comment must state, because it
-- BREAKS the reset-token habit: this token is NOT single-use and does NOT
-- expire. It is the subscriber's permanent capability over their own
-- subscription — the confirm link today, the unsubscribe link at the foot
-- of every future issue. Rotating it would break every unsubscribe link in
-- every mail already sent, which is the one link that must never break.
CREATE TABLE newsletter_subscribers (
    id           BIGINT      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email        TEXT        NOT NULL UNIQUE,
    token_sha256 TEXT        NOT NULL UNIQUE,

    -- The lifecycle is three timestamps, not a status enum: each records
    -- WHEN a fact became true, the combination IS the state, and none of
    -- them lies later. A live recipient is
    -- confirmed_at IS NOT NULL AND unsubscribed_at IS NULL.
    confirmed_at    TIMESTAMPTZ,
    unsubscribed_at TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
