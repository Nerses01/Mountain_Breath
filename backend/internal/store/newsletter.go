package store

import (
	"context"
	"fmt"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// SubscribeNewsletter records a subscription attempt and reports whether a
// confirmation mail should go out. One upsert covers every history the
// address can have:
//
//   - never seen        → new unconfirmed row, mail the link
//   - unconfirmed       → fresh token (the old link dies), mail again
//   - unsubscribed      → fresh token, unconfirm, mail again — coming BACK
//     is a new consent, and the old proof does not transfer
//   - confirmed & live  → change nothing and send NOTHING; a working
//     subscription that re-mails "please confirm" on every form submit is
//     a spam loop with the shop's own name on it
//
// The needsConfirmation return is that last distinction; the HANDLER still
// answers 204 in both cases, so the form is no membership oracle.
func (s *Store) SubscribeNewsletter(ctx context.Context, email, token string) (needsConfirmation bool, err error) {
	err = s.pool.QueryRow(ctx, `
		INSERT INTO newsletter_subscribers (email, token_sha256)
		VALUES ($1, $2)
		ON CONFLICT (email) DO UPDATE SET
		    token_sha256 = CASE
		        WHEN newsletter_subscribers.confirmed_at IS NOT NULL
		         AND newsletter_subscribers.unsubscribed_at IS NULL
		        THEN newsletter_subscribers.token_sha256
		        ELSE EXCLUDED.token_sha256
		    END,
		    confirmed_at = CASE
		        WHEN newsletter_subscribers.unsubscribed_at IS NOT NULL THEN NULL
		        ELSE newsletter_subscribers.confirmed_at
		    END,
		    unsubscribed_at = NULL
		RETURNING confirmed_at IS NULL`,
		email, hashToken(token)).Scan(&needsConfirmation)
	if err != nil {
		return false, fmt.Errorf("subscribing %s: %w", email, err)
	}
	return needsConfirmation, nil
}

// ConfirmNewsletter flips the emailed token's row to confirmed. Idempotent
// on purpose — people re-click links from their inbox, and "you are
// already confirmed" rendered as an ERROR would read as a failure to
// someone who did everything right. Only a token that matches nothing is a
// failure.
func (s *Store) ConfirmNewsletter(ctx context.Context, token string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE newsletter_subscribers
		SET confirmed_at = COALESCE(confirmed_at, now()),
		    unsubscribed_at = NULL
		WHERE token_sha256 = $1`,
		hashToken(token))
	if err != nil {
		return fmt.Errorf("confirming subscription: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UnsubscribeNewsletter is the same capability pointed the other way, and
// equally idempotent: clicking an old unsubscribe link twice is a person
// making sure, not an error.
func (s *Store) UnsubscribeNewsletter(ctx context.Context, token string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE newsletter_subscribers
		SET unsubscribed_at = COALESCE(unsubscribed_at, now())
		WHERE token_sha256 = $1`,
		hashToken(token))
	if err != nil {
		return fmt.Errorf("unsubscribing: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

