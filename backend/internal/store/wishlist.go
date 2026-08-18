package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// ListWishlist returns the user's saved products as full cards (price,
// badge, benefits — everything the grid renders), newest heart first.
// Inactive products drop out of the LIST but keep their rows: a retired
// jar's heart is not the customer's to lose, and it reappears if the
// product does.
//
// A3 wraps each card with WHEN it was hearted. The timestamps come from a
// second small query rather than by threading wishlist columns through the
// shared productCards helper — the helper serves four callers that have no
// saved_at, and two queries beat a parameter that three of four ignore.
func (s *Store) ListWishlist(ctx context.Context, userID int64, view domain.View) ([]domain.WishlistItem, error) {
	products, err := s.productCards(ctx, view, `
		JOIN wishlist_items w ON w.product_id = p.id AND w.user_id = $1
		WHERE p.is_active
		ORDER BY w.added_at DESC, p.id`,
		200, userID)
	if err != nil {
		return nil, err
	}

	rows, err := s.pool.Query(ctx, `
		SELECT product_id, added_at FROM wishlist_items WHERE user_id = $1`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("querying wishlist timestamps: %w", err)
	}
	defer rows.Close()

	savedAt := make(map[int64]time.Time)
	for rows.Next() {
		var productID int64
		var at time.Time
		if err := rows.Scan(&productID, &at); err != nil {
			return nil, fmt.Errorf("scanning wishlist timestamp: %w", err)
		}
		savedAt[productID] = at
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating wishlist timestamps: %w", err)
	}

	items := make([]domain.WishlistItem, 0, len(products))
	for _, p := range products {
		items = append(items, domain.WishlistItem{Product: p, SavedAt: savedAt[p.ID]})
	}
	return items, nil
}

// AddWishlistToCart puts one of each saved, in-stock product into the cart
// (A3) — the reorder merge's sibling, sharing its result contract: one
// transaction, partial success reported line by line. Per product it picks
// the first variant (by id, the stable order) that has room, and merges
// qty 1 into the cart.
func (s *Store) AddWishlistToCart(ctx context.Context, userID int64) (domain.ReorderResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ReorderResult{}, fmt.Errorf("beginning add-all tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT p.id, p.name, v.id, v.label, v.stock_qty, COALESCE(ci.qty, 0)
		FROM wishlist_items w
		JOIN products p ON p.id = w.product_id AND p.is_active
		JOIN product_variants v ON v.product_id = p.id
		LEFT JOIN cart_items ci ON ci.user_id = $1 AND ci.variant_id = v.id
		WHERE w.user_id = $1
		ORDER BY w.added_at DESC, p.id, v.id`,
		userID)
	if err != nil {
		return domain.ReorderResult{}, fmt.Errorf("reading wishlist variants: %w", err)
	}

	type variant struct {
		productID   int64
		productName string
		variantID   int64
		label       string
		stockQty    int
		inCart      int
	}
	var variants []variant
	for rows.Next() {
		var v variant
		if err := rows.Scan(&v.productID, &v.productName, &v.variantID, &v.label,
			&v.stockQty, &v.inCart); err != nil {
			rows.Close()
			return domain.ReorderResult{}, fmt.Errorf("scanning wishlist variant: %w", err)
		}
		variants = append(variants, v)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return domain.ReorderResult{}, fmt.Errorf("iterating wishlist variants: %w", err)
	}

	const maxCartLine = 99 // the cart's own per-line rule, mirrored

	// Group by product preserving order, pick the first variant with room.
	var result domain.ReorderResult
	seen := make(map[int64]bool)
	for _, v := range variants {
		if seen[v.productID] {
			continue
		}
		// Find this product's first variant with room; v itself may not be it.
		var chosen *variant
		for i := range variants {
			c := &variants[i]
			if c.productID == v.productID && c.stockQty-c.inCart > 0 && c.inCart < maxCartLine {
				chosen = c
				break
			}
		}
		seen[v.productID] = true

		if chosen == nil {
			result.Lines = append(result.Lines, domain.ReorderLine{
				Name: v.productName, Issue: domain.ReorderOutOfStock,
			})
			continue
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO cart_items (user_id, variant_id, qty)
			VALUES ($1, $2, 1)
			ON CONFLICT (user_id, variant_id)
			DO UPDATE SET qty = cart_items.qty + 1`,
			userID, chosen.variantID); err != nil {
			return domain.ReorderResult{}, fmt.Errorf("merging wishlist line: %w", err)
		}
		result.Lines = append(result.Lines, domain.ReorderLine{
			Name: chosen.productName, Label: chosen.label, Qty: 1,
		})
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ReorderResult{}, fmt.Errorf("committing add-all: %w", err)
	}
	return result, nil
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
