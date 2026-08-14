package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// ShippingRates loads the standard delivery pricing for every market, keyed
// by currency. One small table, read on every cart view and every checkout —
// and NOT cached in Go, deliberately: the entire reason rates are a table is
// that the family edits them without a deploy, and an in-process cache would
// quietly turn "without a deploy" into "without effect until a restart".
func (s *Store) ShippingRates(ctx context.Context) (map[domain.Currency]domain.ShippingRate, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT currency, base_minor, cold_chain_surcharge_minor, free_over_minor
		FROM shipping_rates
		WHERE method = 'standard'`)
	if err != nil {
		return nil, fmt.Errorf("querying shipping rates: %w", err)
	}
	defer rows.Close()

	rates := make(map[domain.Currency]domain.ShippingRate)
	for rows.Next() {
		var currency string
		var r domain.ShippingRate
		if err := rows.Scan(&currency, &r.BaseMinor, &r.ColdChainSurchargeMinor, &r.FreeOverMinor); err != nil {
			return nil, fmt.Errorf("scanning shipping rate: %w", err)
		}
		rates[domain.Currency(currency)] = r
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating shipping rates: %w", err)
	}
	return rates, nil
}

// DefaultAddress returns the user's saved address for pre-filling the
// checkout form, or ErrNotFound for a first-time customer — which the API
// maps to an empty form, not an error page.
func (s *Store) DefaultAddress(ctx context.Context, userID int64) (domain.Address, error) {
	var a domain.Address
	err := s.pool.QueryRow(ctx, `
		SELECT first_name, last_name, phone, street, city, postal_code, country
		FROM addresses
		WHERE user_id = $1 AND is_default`,
		userID,
	).Scan(&a.FirstName, &a.LastName, &a.Phone, &a.Street, &a.City, &a.PostalCode, &a.Country)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Address{}, domain.ErrNotFound
		}
		return domain.Address{}, fmt.Errorf("querying default address: %w", err)
	}
	return a, nil
}

// upsertDefaultAddress saves the address a checkout used as the user's
// default, inside the checkout's own transaction. The partial unique index
// (one default per user) is what makes ON CONFLICT work here: the index is
// a valid arbiter for the upsert, so "insert or update the default" is one
// statement with no read-then-write race.
func upsertDefaultAddress(ctx context.Context, tx pgx.Tx, userID int64, a domain.Address) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO addresses (user_id, first_name, last_name, phone, street,
		                       city, postal_code, country, is_default)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE)
		ON CONFLICT (user_id) WHERE is_default DO UPDATE SET
		    first_name = EXCLUDED.first_name,
		    last_name = EXCLUDED.last_name,
		    phone = EXCLUDED.phone,
		    street = EXCLUDED.street,
		    city = EXCLUDED.city,
		    postal_code = EXCLUDED.postal_code,
		    country = EXCLUDED.country,
		    updated_at = now()`,
		userID, a.FirstName, a.LastName, a.Phone, a.Street, a.City, a.PostalCode, a.Country)
	if err != nil {
		return fmt.Errorf("saving default address: %w", err)
	}
	return nil
}
