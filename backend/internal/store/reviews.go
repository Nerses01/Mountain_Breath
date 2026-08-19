package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// recomputeRating rewrites one product's aggregate FROM the reviews table.
//
// This is the heart of E4, and the choice worth understanding is what it
// does NOT do: it never nudges the stored average by the delta of one review.
// Incremental updates are cheaper and they drift — every floating-point
// addition and subtraction loses a little, and an aggregate that disagrees
// with the rows it summarises is a bug nobody notices until a customer does.
// Recomputing reads only this product's reviews, which is bounded and small,
// and is exact by construction.
//
// It takes a tx rather than the pool because it is never correct on its own:
// the aggregate must move in the SAME transaction as the review that moved
// it, or a reader can see one without the other.
//
// AVG over zero rows is NULL, not 0 — coalesce, or the NOT NULL column
// rejects the write the moment a product's last review is rejected.
func recomputeRating(ctx context.Context, tx pgx.Tx, productID int64) error {
	_, err := tx.Exec(ctx, `
		UPDATE products p
		SET rating_avg   = COALESCE(r.avg_rating, 0),
		    rating_count = COALESCE(r.n, 0)
		FROM (
		    SELECT avg(rating)::numeric(3,2) AS avg_rating, count(*) AS n
		    FROM reviews
		    -- Only PUBLISHED reviews count. A pending review must not move
		    -- the public average before a human has looked at it, and that
		    -- is exactly why moderation has to trigger a recompute too.
		    WHERE product_id = $1 AND status = 'published'
		) r
		WHERE p.id = $1`,
		productID)
	if err != nil {
		return fmt.Errorf("recomputing rating for product %d: %w", productID, err)
	}
	return nil
}

// CanReview reports whether a user has standing to review a product: they
// must have a DELIVERED order containing one of its variants.
//
// EXISTS rather than a count or a join: the question is yes/no, and EXISTS
// stops at the first matching row instead of finding every order the
// customer ever placed for it.
func (s *Store) CanReview(ctx context.Context, userID int64, productSlug string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1
		    FROM orders o
		    JOIN order_items oi ON oi.order_id = o.id
		    JOIN product_variants v ON v.id = oi.variant_id
		    JOIN products p ON p.id = v.product_id
		    WHERE o.user_id = $1
		      AND o.status = $3
		      AND p.slug = $2
		)
		-- ...and has not already reviewed it. Both halves of "may I write
		-- one" answered in a single round trip, so the UI never shows a form
		-- that the write path will refuse.
		AND NOT EXISTS (
		    SELECT 1 FROM reviews r
		    JOIN products p2 ON p2.id = r.product_id
		    WHERE r.user_id = $1 AND p2.slug = $2
		)`,
		userID, productSlug, domain.OrderDelivered,
	).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("checking review eligibility: %w", err)
	}
	return ok, nil
}

// CreateReview stores a review and moves the product's aggregate, in one
// transaction.
//
// The verified-purchase rule is re-checked HERE rather than trusted from the
// handler: the API's can_review is a hint for rendering, and a hint is not a
// permission. Anyone can POST.
func (s *Store) CreateReview(ctx context.Context, r *domain.Review, productSlug string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning review tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Resolve the slug inside the transaction so the product cannot be
	// deleted between the lookup and the insert.
	err = tx.QueryRow(ctx,
		`SELECT id FROM products WHERE slug = $1 AND is_active`, productSlug).Scan(&r.ProductID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("resolving product %q: %w", productSlug, err)
	}

	var purchased bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1
		    FROM orders o
		    JOIN order_items oi ON oi.order_id = o.id
		    JOIN product_variants v ON v.id = oi.variant_id
		    WHERE o.user_id = $1 AND o.status = $3 AND v.product_id = $2
		)`,
		r.UserID, r.ProductID, domain.OrderDelivered).Scan(&purchased)
	if err != nil {
		return fmt.Errorf("checking purchase: %w", err)
	}
	if !purchased {
		return domain.ErrNotPurchased
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO reviews (product_id, user_id, rating, title, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at`,
		r.ProductID, r.UserID, r.Rating, r.Title, r.Body,
	).Scan(&r.ID, &r.Status, &r.CreatedAt)
	if err != nil {
		// The UNIQUE constraint, not an application check, is what makes
		// "one review per person per product" true under concurrency.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.ErrAlreadyReviewed
		}
		return fmt.Errorf("inserting review: %w", err)
	}

	// A new review arrives `pending`, so the public average does not move
	// yet — but recomputing anyway keeps ONE rule ("any write to reviews
	// recomputes") instead of a second rule about which writes are exempt.
	if err := recomputeRating(ctx, tx, r.ProductID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing review: %w", err)
	}
	return nil
}

// UpdateReviewStatus is the moderation action: publish or reject.
func (s *Store) UpdateReviewStatus(ctx context.Context, reviewID int64, status string) (domain.Review, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.Review{}, fmt.Errorf("beginning moderation tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var r domain.Review
	err = tx.QueryRow(ctx, `
		UPDATE reviews
		SET status = $2, updated_at = now()
		WHERE id = $1
		RETURNING id, product_id, user_id, rating, title, body, status, created_at`,
		reviewID, status,
	).Scan(&r.ID, &r.ProductID, &r.UserID, &r.Rating, &r.Title, &r.Body, &r.Status, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Review{}, domain.ErrNotFound
		}
		return domain.Review{}, fmt.Errorf("updating review status: %w", err)
	}

	// The reason moderation is a transaction at all: publishing or rejecting
	// changes what the public average is made of, so the aggregate has to
	// move with it. "Moderation changes the public average immediately" is
	// this line.
	if err := recomputeRating(ctx, tx, r.ProductID); err != nil {
		return domain.Review{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Review{}, fmt.Errorf("committing moderation: %w", err)
	}
	return r, nil
}

// ListReviews returns one page of reviews plus the total.
//
// The public caller sets Status to published; the admin queue sets whatever
// it is filtering by. An empty Status means "any", which is why the handler
// — not this function — is responsible for never leaving it empty on a
// public path.
func (s *Store) ListReviews(ctx context.Context, f domain.ReviewFilter) ([]domain.Review, int, error) {
	const from = `
		FROM reviews r
		JOIN products p ON p.id = r.product_id
		JOIN users u ON u.id = r.user_id
		WHERE ($1 = '' OR p.slug = $1)
		  AND ($2 = '' OR r.status = $2)`

	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.product_id, r.user_id, r.rating, r.title, r.body,
		       r.status, r.created_at, u.email`+from+`
		ORDER BY r.created_at DESC, r.id DESC
		LIMIT $3 OFFSET $4`,
		f.ProductSlug, f.Status, f.PerPage, f.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("querying reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]domain.Review, 0)
	for rows.Next() {
		var r domain.Review
		if err := rows.Scan(&r.ID, &r.ProductID, &r.UserID, &r.Rating, &r.Title,
			&r.Body, &r.Status, &r.CreatedAt, &r.AuthorEmail); err != nil {
			return nil, 0, fmt.Errorf("scanning review: %w", err)
		}
		reviews = append(reviews, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating reviews: %w", err)
	}

	var total int
	if err := s.pool.QueryRow(ctx, `SELECT count(*)`+from,
		f.ProductSlug, f.Status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting reviews: %w", err)
	}
	return reviews, total, nil
}

// ReviewsByUser is the data view's read (F2, decision #97): everything
// this person has written, every status included — pending and rejected
// reviews are still THEIR words, and an export that hid them would be
// lying by omission. The parallel slug slice is the ListUsers pairing
// idiom: the export wants a human-readable product reference without
// teaching domain.Review a field only this caller uses.
func (s *Store) ReviewsByUser(ctx context.Context, userID int64) ([]domain.Review, []string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.product_id, r.rating, r.title, r.body, r.status, r.created_at,
		       p.slug
		FROM reviews r
		JOIN products p ON p.id = r.product_id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC`,
		userID)
	if err != nil {
		return nil, nil, fmt.Errorf("querying user reviews: %w", err)
	}
	defer rows.Close()

	reviews := make([]domain.Review, 0)
	slugs := make([]string, 0)
	for rows.Next() {
		var r domain.Review
		var slug string
		if err := rows.Scan(&r.ID, &r.ProductID, &r.Rating, &r.Title, &r.Body,
			&r.Status, &r.CreatedAt, &slug); err != nil {
			return nil, nil, fmt.Errorf("scanning user review: %w", err)
		}
		r.UserID = userID
		reviews = append(reviews, r)
		slugs = append(slugs, slug)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterating user reviews: %w", err)
	}
	return reviews, slugs, nil
}
