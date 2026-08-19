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

// GetUserByID is GetUserByEmail's sibling read, added for F2's status
// mailer: an order knows its customer only as user_id, and the mail needs
// the email plus the notify_order_updates toggle that gates it.
func (s *Store) GetUserByID(ctx context.Context, userID int64) (domain.User, error) {
	var u domain.User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, role, created_at,
		       full_name, phone, notify_order_updates
		FROM users
		WHERE id = $1`,
		userID,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt,
		&u.FullName, &u.Phone, &u.NotifyOrderUpdates)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, fmt.Errorf("querying user by id: %w", err)
	}
	return u, nil
}

// ── F2: user administration (decision #96) ────────────────────────────────

// ListUsers is the admin table's read: everyone, newest first, with a
// per-user order count for context (all orders, cancelled included — the
// column answers "how much history does this account have", not revenue).
func (s *Store) ListUsers(ctx context.Context) ([]domain.User, []int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT u.id, u.email, u.role, u.created_at, u.full_name,
		       (SELECT count(*) FROM orders o WHERE o.user_id = u.id)::int
		FROM users u
		ORDER BY u.created_at DESC, u.id DESC`)
	if err != nil {
		return nil, nil, fmt.Errorf("querying users: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	counts := make([]int, 0)
	for rows.Next() {
		var u domain.User
		var n int
		if err := rows.Scan(&u.ID, &u.Email, &u.Role, &u.CreatedAt, &u.FullName, &n); err != nil {
			return nil, nil, fmt.Errorf("scanning user: %w", err)
		}
		users = append(users, u)
		counts = append(counts, n)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating users: %w", err)
	}
	return users, counts, nil
}

// UpdateUserRole promotes or demotes — with the shop's one role invariant
// enforced where it can actually hold: "at least one admin" is a COUNT,
// so like the promo cap it cannot be an index or a CHECK; it is counted
// under locks. The transaction locks every current admin row (ordered, so
// two concurrent demotions cannot deadlock) and only then counts: the
// second of two racing demotions waits, re-reads a world with one admin
// left, and is refused. Promotion never locks more than the target — the
// count only ever grows on that path.
func (s *Store) UpdateUserRole(ctx context.Context, userID int64, role string) (domain.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("beginning role tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if role == domain.RoleCustomer {
		// The demote path: lock the admin set, then count it. FOR UPDATE
		// re-evaluates each row after a lock wait (Postgres's EPQ), so a
		// row another transaction just demoted no longer matches and drops
		// out of OUR set — the count below is post-wait truth, not a
		// snapshot from before the race.
		adminRows, err := tx.Query(ctx, `
			SELECT id FROM users WHERE role = 'admin' ORDER BY id FOR UPDATE`)
		if err != nil {
			return domain.User{}, fmt.Errorf("locking admin rows: %w", err)
		}
		admins := make(map[int64]bool)
		for adminRows.Next() {
			var id int64
			if err := adminRows.Scan(&id); err != nil {
				adminRows.Close()
				return domain.User{}, fmt.Errorf("scanning admin row: %w", err)
			}
			admins[id] = true
		}
		adminRows.Close()
		if err := adminRows.Err(); err != nil {
			return domain.User{}, fmt.Errorf("iterating admin rows: %w", err)
		}

		if admins[userID] && len(admins) == 1 {
			return domain.User{}, domain.ErrLastAdmin
		}
	}

	tag, err := tx.Exec(ctx,
		`UPDATE users SET role = $2 WHERE id = $1`, userID, role)
	if err != nil {
		return domain.User{}, fmt.Errorf("updating role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.User{}, domain.ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("committing role change: %w", err)
	}
	return s.GetUserByID(ctx, userID)
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
