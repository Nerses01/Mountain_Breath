package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// Postgres error code: foreign key violation.
const foreignKeyViolation = "23503"

// GetCart resolves product names in the requested language, like every other
// storefront read.
//
// It did NOT until E2, and the omission is worth recording: E1.5 translated
// the catalog and left the cart on products.name, so an Armenian visitor
// browsed in Armenian and then found English names in their basket. Nothing
// failed — the fallback chain hands back perfectly valid English — which is
// exactly why a missing translation join is easy to ship. Order items are
// deliberately NOT translated the same way: those are snapshots of what the
// customer actually bought, frozen at checkout.
// E5 gave it a currency for the same reason and with the same shape as
// attachVariants: one row per (line, currency), grouped back in Go, so the
// basket can show both markets without converting either.
func (s *Store) GetCart(ctx context.Context, userID int64, view domain.View) ([]domain.CartItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ci.variant_id, COALESCE(t.name, en.name, p.name), p.slug,
		       v.label, v.stock_qty, ci.qty,
		       ep.currency, ep.price_minor,
		       res.price_minor AS resolved_minor
		FROM cart_items ci
		JOIN product_variants v ON v.id = ci.variant_id
		JOIN products p ON p.id = v.product_id
		JOIN variant_effective_prices ep ON ep.variant_id = v.id
		LEFT JOIN variant_effective_prices res
		       ON res.variant_id = v.id AND res.currency = $3
		LEFT JOIN product_translations t  ON t.product_id  = p.id AND t.locale  = $2
		LEFT JOIN product_translations en ON en.product_id = p.id AND en.locale = 'en'
		WHERE ci.user_id = $1
		ORDER BY ci.added_at, ci.variant_id, ep.currency`,
		userID, view.EffectiveLocale(), view.EffectiveCurrency())
	if err != nil {
		return nil, fmt.Errorf("querying cart: %w", err)
	}
	defer rows.Close()

	items := make([]domain.CartItem, 0)
	var current *domain.CartItem
	for rows.Next() {
		var (
			variantID, priceMinor            int64
			name, slug, label, currency      string
			stockQty, qty                    int
			resolvedMinor                    *int64
		)
		if err := rows.Scan(&variantID, &name, &slug, &label, &stockQty, &qty,
			&currency, &priceMinor, &resolvedMinor); err != nil {
			return nil, fmt.Errorf("scanning cart row: %w", err)
		}

		if current == nil || current.VariantID != variantID {
			items = append(items, domain.CartItem{
				VariantID: variantID, ProductName: name, ProductSlug: slug,
				Label: label, StockQty: stockQty, Qty: qty,
				Prices: make(domain.Money, len(domain.Currencies)),
			})
			current = &items[len(items)-1]
			if resolvedMinor != nil {
				current.PriceMinor = *resolvedMinor
			}
		}
		current.Prices[domain.Currency(currency)] = priceMinor
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating cart rows: %w", err)
	}
	return items, nil
}

// SetCartItem sets the quantity for a variant in the user's cart (upsert).
func (s *Store) SetCartItem(ctx context.Context, userID, variantID int64, qty int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO cart_items (user_id, variant_id, qty)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, variant_id) DO UPDATE SET qty = EXCLUDED.qty`,
		userID, variantID, qty)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
			// variant_id points at nothing → the variant doesn't exist
			return domain.ErrNotFound
		}
		return fmt.Errorf("upserting cart item: %w", err)
	}
	return nil
}

func (s *Store) DeleteCartItem(ctx context.Context, userID, variantID int64) error {
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM cart_items WHERE user_id = $1 AND variant_id = $2`,
		userID, variantID); err != nil {
		return fmt.Errorf("deleting cart item: %w", err)
	}
	return nil
}
