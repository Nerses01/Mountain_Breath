package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// ListWishlist returns the user's saved products as full cards (price,
// badge, benefits — everything the grid renders), newest heart first.
// Inactive products drop out of the LIST but keep their rows: a retired
// jar's heart is not the customer's to lose, and it reappears if the
// product does.
func (s *Store) ListWishlist(ctx context.Context, userID int64, view domain.View) ([]domain.Product, error) {
	return s.productCards(ctx, view, `
		JOIN wishlist_items w ON w.product_id = p.id AND w.user_id = $1
		WHERE p.is_active
		ORDER BY w.added_at DESC, p.id`,
		200, userID)
}

// AddWishlistItem hearts a product — idempotently, like every set-shaped
// write in this API: hearting twice is the same fact stated twice, so the
// second write is a no-op, not an error.
func (s *Store) AddWishlistItem(ctx context.Context, userID, productID int64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO wishlist_items (user_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, product_id) DO NOTHING`,
		userID, productID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
			return domain.ErrNotFound
		}
		return fmt.Errorf("adding wishlist item: %w", err)
	}
	return nil
}

// RemoveWishlistItem un-hearts. Removing an absent heart succeeds — the
// state the customer asked for is the state they get.
func (s *Store) RemoveWishlistItem(ctx context.Context, userID, productID int64) error {
	if _, err := s.pool.Exec(ctx, `
		DELETE FROM wishlist_items WHERE user_id = $1 AND product_id = $2`,
		userID, productID); err != nil {
		return fmt.Errorf("removing wishlist item: %w", err)
	}
	return nil
}

// SaveForLater moves a cart LINE to the wishlist — one transaction, because
// "move" means the line must never exist in both places or neither. The
// grain deliberately changes in transit: a cart line is a variant with a
// quantity ("2 × 500 g"), a wishlist entry is just the product — later-you
// wants to remember the jar, not the jar size and count of a Tuesday.
func (s *Store) SaveForLater(ctx context.Context, userID, variantID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning save-for-later tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// DELETE … RETURNING is the read and the write in one statement: it
	// removes the line and tells us which product it was, and zero rows
	// back means there was no such line to move.
	var productID int64
	err = tx.QueryRow(ctx, `
		DELETE FROM cart_items ci
		USING product_variants v
		WHERE ci.user_id = $1 AND ci.variant_id = $2 AND v.id = ci.variant_id
		RETURNING v.product_id`,
		userID, variantID).Scan(&productID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("removing cart line: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO wishlist_items (user_id, product_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, product_id) DO NOTHING`,
		userID, productID); err != nil {
		return fmt.Errorf("inserting wishlist item: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing save-for-later: %w", err)
	}
	return nil
}
