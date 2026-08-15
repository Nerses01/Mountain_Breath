package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// FindOrCreateOAuthUser resolves a provider identity to a shop account,
// in strict preference order:
//
//  1. the identity is KNOWN (provider, subject) → that account, done. The
//     subject is the identity; the email is not even looked at, because
//     providers let people change email and the account must not follow.
//  2. an account already exists under the provider-VERIFIED email → link:
//     insert the identity onto it. This is the "I registered with a
//     password last year, now I clicked the Google button" case, and it
//     must not mint a duplicate account. The caller guarantees the email
//     was verified by the provider — linking on an unverified email would
//     let anyone claim any account by registering it at the provider.
//  3. nobody → a new customer account with password_hash '' (bcrypt matches
//     nothing against an empty hash, so password login fails closed until
//     forgot-password sets a real one).
//
// One transaction, so a crash between "create user" and "create identity"
// cannot strand a passwordless account nothing can sign into.
func (s *Store) FindOrCreateOAuthUser(ctx context.Context, provider, subject, email string) (domain.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, fmt.Errorf("beginning oauth tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var u domain.User

	// 1. Known identity.
	err = tx.QueryRow(ctx, `
		SELECT u.id, u.email, u.password_hash, u.role, u.created_at
		FROM oauth_identities oi
		JOIN users u ON u.id = oi.user_id
		WHERE oi.provider = $1 AND oi.subject = $2`,
		provider, subject).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	switch {
	case err == nil:
		return u, tx.Commit(ctx)
	case !errors.Is(err, pgx.ErrNoRows):
		return domain.User{}, fmt.Errorf("querying oauth identity: %w", err)
	}

	// 2 & 3. Find the account by verified email, or mint one.
	err = tx.QueryRow(ctx, `
		SELECT id, email, password_hash, role, created_at
		FROM users WHERE email = $1`,
		email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		u = domain.User{Email: email, PasswordHash: "", Role: domain.RoleCustomer}
		err = tx.QueryRow(ctx, `
			INSERT INTO users (email, password_hash, role)
			VALUES ($1, '', $2)
			RETURNING id, created_at`,
			email, u.Role).Scan(&u.ID, &u.CreatedAt)
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("resolving oauth account: %w", err)
	}

	// ON CONFLICT DO NOTHING: two parallel first sign-ins race here, and the
	// loser's insert finding the row already present is success, not error —
	// both transactions resolved the same person to the same account.
	if _, err := tx.Exec(ctx, `
		INSERT INTO oauth_identities (user_id, provider, subject)
		VALUES ($1, $2, $3)
		ON CONFLICT (provider, subject) DO NOTHING`,
		u.ID, provider, subject); err != nil {
		return domain.User{}, fmt.Errorf("linking oauth identity: %w", err)
	}

	return u, tx.Commit(ctx)
}
