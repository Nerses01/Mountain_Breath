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

func (s *Store) GetCart(ctx context.Context, userID int64) ([]domain.CartItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ci.variant_id, p.name, p.slug, v.label, v.price_minor, v.stock_qty, ci.qty
		FROM cart_items ci
		JOIN product_variants v ON v.id = ci.variant_id
		JOIN products p ON p.id = v.product_id
		WHERE ci.user_id = $1
		ORDER BY ci.added_at`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("querying cart: %w", err)
	}
	defer rows.Close()

	items := make([]domain.CartItem, 0)
	for rows.Next() {
		var it domain.CartItem
		if err := rows.Scan(&it.VariantID, &it.ProductName, &it.ProductSlug,
			&it.Label, &it.PriceMinor, &it.StockQty, &it.Qty); err != nil {
			return nil, fmt.Errorf("scanning cart row: %w", err)
		}
		items = append(items, it)
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
