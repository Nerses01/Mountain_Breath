package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// CreatePasswordReset stores the SHA-256 of a fresh reset token (the raw
// token exists only in the email being sent) and retires the user's earlier
// unused tokens — the newest link is the only live one, so a customer who
// clicks "send again" three times cannot be phished with link number one a
// week later.
func (s *Store) CreatePasswordReset(ctx context.Context, userID int64, token string, expiresAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning reset tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		UPDATE password_reset_tokens
		SET used_at = now()
		WHERE user_id = $1 AND used_at IS NULL`,
		userID); err != nil {
		return fmt.Errorf("retiring old reset tokens: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO password_reset_tokens (user_id, token_sha256, expires_at)
		VALUES ($1, $2, $3)`,
		userID, hashToken(token), expiresAt); err != nil {
		return fmt.Errorf("inserting reset token: %w", err)
	}
	return tx.Commit(ctx)
}

// ConsumePasswordReset spends a token and installs the new password hash —
// one transaction, because "single use" is a property only a transaction
// can promise: two racing submits both find the token unused unless the
// first locks it (FOR UPDATE) before the second reads.
//
// It also deletes EVERY session the user has. A password reset usually
// means "someone may have my credentials", and a reset that leaves a
// stolen session alive has changed the lock while leaving a window open.
// The customer signs in once with the new password; the thief signs in
// never.
func (s *Store) ConsumePasswordReset(ctx context.Context, token, newPasswordHash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning consume tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var userID int64
	err = tx.QueryRow(ctx, `
		SELECT user_id FROM password_reset_tokens
		WHERE token_sha256 = $1 AND used_at IS NULL AND expires_at > now()
		FOR UPDATE`,
		hashToken(token)).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// Unknown, spent and expired all answer identically — the
			// caller (and therefore the visitor) learns only "this link no
			// longer works", never which kind of no.
			return domain.ErrNotFound
		}
		return fmt.Errorf("locking reset token: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE password_reset_tokens SET used_at = now()
		WHERE token_sha256 = $1`,
		hashToken(token)); err != nil {
		return fmt.Errorf("spending reset token: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $1 WHERE id = $2`,
		newPasswordHash, userID); err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM sessions WHERE user_id = $1`,
		userID); err != nil {
		return fmt.Errorf("revoking sessions: %w", err)
	}
	return tx.Commit(ctx)
}
