package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

func (s *Store) CreateUser(ctx context.Context, u *domain.User) error {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`,
		u.Email, u.PasswordHash, u.Role,
	).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.ErrEmailTaken
		}
		return fmt.Errorf("inserting user: %w", err)
	}
	return nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, created_at,
		       full_name, phone, notify_order_updates
		FROM users
		WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt,
		&u.FullName, &u.Phone, &u.NotifyOrderUpdates)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("querying user by email: %w", err)
	}
	return u, nil
}

// UpdateProfile writes the settings form's two fields (A5). Whole-value
// semantics like every PUT-shaped write here: the form shows both fields,
// so it sends both.
func (s *Store) UpdateProfile(ctx context.Context, userID int64, fullName, phone string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users SET full_name = $2, phone = $3 WHERE id = $1`,
		userID, fullName, phone)
	if err != nil {
		return fmt.Errorf("updating profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// SetNotifyOrderUpdates flips the one wired notification preference (A5,
// decision log #87). F2's mailer reads it before sending.
func (s *Store) SetNotifyOrderUpdates(ctx context.Context, userID int64, on bool) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE users SET notify_order_updates = $2 WHERE id = $1`,
		userID, on); err != nil {
		return fmt.Errorf("updating notification preference: %w", err)
	}
	return nil
}

// ChangePassword rotates the hash and revokes every OTHER session in the
// same transaction (A5). The difference from the reset flow's revoke-ALL is
// deliberate: a reset means "someone may hold my password" (kill every
// session, including the thief's); a signed-in change is the owner at the
// keyboard — logging THEM out too would punish the person doing the right
// thing. keepToken is the caller's own session, identified the same way
// sessions always are: by its hash, never the raw token.
func (s *Store) ChangePassword(ctx context.Context, userID int64, newHash, keepToken string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning change-password tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $2 WHERE id = $1`,
		userID, newHash)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM sessions WHERE user_id = $1 AND token_hash <> $2`,
		userID, hashToken(keepToken)); err != nil {
		return fmt.Errorf("revoking other sessions: %w", err)
	}
	return tx.Commit(ctx)
}
