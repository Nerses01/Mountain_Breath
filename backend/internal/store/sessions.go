package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The DB stores only the SHA-256 of the token; the raw token lives solely in
// the user's cookie. SHA-256 (not bcrypt) is fine here: tokens are 256-bit
// random values, not guessable passwords.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Store) CreateSession(ctx context.Context, token string, userID int64, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)`,
		hashToken(token), userID, expiresAt)
	if err != nil {
		return fmt.Errorf("inserting session: %w", err)
	}
	return nil
}

// GetUserBySession resolves a raw cookie token to its user, if the session
// exists and has not expired. Expiry is checked in SQL — one source of truth
// for "now".
func (s *Store) GetUserBySession(ctx context.Context, token string) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.password_hash, u.role, u.created_at,
		       u.full_name, u.phone, u.notify_order_updates
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`,
		hashToken(token),
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt,
		&u.FullName, &u.Phone, &u.NotifyOrderUpdates)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("querying session: %w", err)
	}
	return u, nil
}

func (s *Store) DeleteSession(ctx context.Context, token string) error {
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM sessions WHERE token_hash = $1`, hashToken(token)); err != nil {
		return fmt.Errorf("deleting session: %w", err)
	}
	return nil
}
