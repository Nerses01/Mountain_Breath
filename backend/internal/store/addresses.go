package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The account page's address book (E8). E6 built the table and scoped it to
// one default row per user (the checkout upserts it); these methods are the
// "several named addresses" half the plan deferred here.
//
// Every statement filters by user_id AS WELL AS id. That is not belt and
// braces — it is the authorization: address 17 belonging to someone else
// must behave exactly like address 17 not existing, and putting the rule in
// the WHERE clause means no code path can forget it.

const addressColumns = `id, label, is_default, leave_with_neighbour,
	first_name, last_name, phone, street, city, postal_code, country`

func scanAddressEntry(row pgx.Row) (domain.AddressEntry, error) {
	var e domain.AddressEntry
	err := row.Scan(&e.ID, &e.Label, &e.IsDefault, &e.LeaveWithNeighbour,
		&e.FirstName, &e.LastName, &e.Phone, &e.Street,
		&e.City, &e.PostalCode, &e.Country)
	return e, err
}

func (s *Store) ListAddresses(ctx context.Context, userID int64) ([]domain.AddressEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+addressColumns+`
		FROM addresses
		WHERE user_id = $1
		ORDER BY is_default DESC, id`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("querying addresses: %w", err)
	}
	defer rows.Close()

	entries := make([]domain.AddressEntry, 0)
	for rows.Next() {
		e, err := scanAddressEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("scanning address: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating addresses: %w", err)
	}
	return entries, nil
}

// CreateAddress inserts a book entry. The first address a user saves
// becomes the default whatever the flag said — a book with entries but no
// default would leave the checkout prefill with nothing to stand on — and
// an explicit new default clears the old one first, inside the same
// transaction, because the partial unique index (one default per user)
// rejects the intermediate state (the 000011 one-primary-image lesson).
func (s *Store) CreateAddress(ctx context.Context, userID int64, e domain.AddressEntry) (domain.AddressEntry, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.AddressEntry{}, fmt.Errorf("beginning create-address tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var hasAny bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM addresses WHERE user_id = $1)`,
		userID).Scan(&hasAny); err != nil {
		return domain.AddressEntry{}, fmt.Errorf("checking address book: %w", err)
	}
	e.IsDefault = e.IsDefault || !hasAny

	if e.IsDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE addresses SET is_default = FALSE
			WHERE user_id = $1 AND is_default`,
			userID); err != nil {
			return domain.AddressEntry{}, fmt.Errorf("clearing old default: %w", err)
		}
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO addresses (user_id, label, is_default, leave_with_neighbour,
		                       first_name, last_name, phone, street,
		                       city, postal_code, country)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		userID, e.Label, e.IsDefault, e.LeaveWithNeighbour,
		e.FirstName, e.LastName, e.Phone, e.Street,
		e.City, e.PostalCode, e.Country,
	).Scan(&e.ID)
	if err != nil {
		return domain.AddressEntry{}, fmt.Errorf("inserting address: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.AddressEntry{}, fmt.Errorf("committing create-address: %w", err)
	}
	return e, nil
}

// UpdateAddress rewrites one entry, including its default flag. Making an
// entry the default demotes the previous one; taking the flag AWAY from
// the default is allowed and leaves the book without one — the checkout
// then simply has nothing to prefill, which is a true statement about the
// customer's choices.
func (s *Store) UpdateAddress(ctx context.Context, userID int64, e domain.AddressEntry) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning update-address tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if e.IsDefault {
		if _, err := tx.Exec(ctx, `
			UPDATE addresses SET is_default = FALSE
			WHERE user_id = $1 AND is_default AND id <> $2`,
			userID, e.ID); err != nil {
			return fmt.Errorf("clearing old default: %w", err)
		}
	}

	tag, err := tx.Exec(ctx, `
		UPDATE addresses
		SET label = $3, is_default = $4, leave_with_neighbour = $5,
		    first_name = $6, last_name = $7, phone = $8, street = $9,
		    city = $10, postal_code = $11, country = $12,
		    updated_at = now()
		WHERE id = $1 AND user_id = $2`,
		e.ID, userID, e.Label, e.IsDefault, e.LeaveWithNeighbour,
		e.FirstName, e.LastName, e.Phone, e.Street,
		e.City, e.PostalCode, e.Country)
	if err != nil {
		return fmt.Errorf("updating address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return tx.Commit(ctx)
}

func (s *Store) DeleteAddress(ctx context.Context, userID, addressID int64) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM addresses WHERE id = $1 AND user_id = $2`,
		addressID, userID)
	if err != nil {
		return fmt.Errorf("deleting address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
