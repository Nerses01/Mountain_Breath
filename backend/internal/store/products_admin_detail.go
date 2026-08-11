package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The admin write paths for everything E3 added to a product page.
//
// A note that applies to all of them: these collections are REPLACED, not
// patched. The rows are keyed by position (highlights, usage cards) or
// curated as a list (related products), so "row 3" has no stable identity
// once the editor reorders or removes one. Delete-then-insert inside a
// transaction is simpler than diffing, and — because it is one transaction —
// there is no moment where the product has half its bullets.

// AddProductImage appends an upload to a product's gallery.
//
// The first image a product ever gets becomes its primary automatically. A
// gallery whose hero is unset renders nothing where the design's 520px hero
// should be, and making the admin remember one extra click to avoid that is
// a rule the database can apply itself.
func (s *Store) AddProductImage(ctx context.Context, productID int64, url string, alts map[domain.Locale]string) (domain.ProductImage, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return domain.ProductImage{}, fmt.Errorf("beginning add-image tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var img domain.ProductImage
	img.URL = url
	err = tx.QueryRow(ctx, `
		INSERT INTO product_images (product_id, url, sort_order, is_primary)
		SELECT $1, $2,
		       COALESCE(max(sort_order) + 1, 0),
		       -- No rows yet ⇒ this one is the hero. count(*) over the same
		       -- filtered set the max comes from, so both read one scan.
		       count(*) = 0
		FROM product_images WHERE product_id = $1
		RETURNING id, sort_order, is_primary`,
		productID, url,
	).Scan(&img.ID, &img.SortOrder, &img.IsPrimary)
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == foreignKeyViolation {
			return domain.ProductImage{}, domain.ErrNotFound
		}
		return domain.ProductImage{}, fmt.Errorf("inserting product image: %w", err)
	}

	if err := upsertImageAlts(ctx, tx, img.ID, alts); err != nil {
		return domain.ProductImage{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.ProductImage{}, fmt.Errorf("committing add-image: %w", err)
	}
	return img, nil
}

func upsertImageAlts(ctx context.Context, tx pgx.Tx, imageID int64, alts map[domain.Locale]string) error {
	for locale, alt := range alts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_image_translations (image_id, locale, alt)
			VALUES ($1, $2, $3)
			ON CONFLICT (image_id, locale) DO UPDATE SET alt = EXCLUDED.alt`,
			imageID, locale, alt,
		); err != nil {
			return fmt.Errorf("upserting %s image alt: %w", locale, err)
		}
	}
	return nil
}

// SaveProductImages applies a reorder, a new hero and edited alt text in one
// transaction.
//
// The primary flag is cleared for the WHOLE product before the new one is
// set. That is not tidiness: the partial unique index in 000011 rejects two
// primary rows, so setting the new one first would fail against the old one
// still being true. Two statements in one transaction, in that order, is the
// only sequence the constraint allows.
func (s *Store) SaveProductImages(ctx context.Context, productID int64, images []domain.ProductImage, alts map[int64]map[domain.Locale]string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning save-images tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE product_images SET is_primary = FALSE WHERE product_id = $1`,
		productID); err != nil {
		return fmt.Errorf("clearing primary image: %w", err)
	}

	for _, img := range images {
		// WHERE product_id too, not just id: without it a caller could
		// reorder another product's images by guessing an id.
		tag, err := tx.Exec(ctx, `
			UPDATE product_images
			SET sort_order = $1, is_primary = $2
			WHERE id = $3 AND product_id = $4`,
			img.SortOrder, img.IsPrimary, img.ID, productID)
		if err != nil {
			return fmt.Errorf("updating image %d: %w", img.ID, err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrNotFound
		}
		if err := upsertImageAlts(ctx, tx, img.ID, alts[img.ID]); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing save-images: %w", err)
	}
	return nil
}

// DeleteProductImage removes one photo, and promotes another to hero if it
// was the primary — otherwise deleting the hero leaves a gallery that renders
// no hero at all, which is the same hole AddProductImage's auto-primary
// closes from the other side.
func (s *Store) DeleteProductImage(ctx context.Context, productID, imageID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning delete-image tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var wasPrimary bool
	err = tx.QueryRow(ctx, `
		DELETE FROM product_images
		WHERE id = $1 AND product_id = $2
		RETURNING is_primary`,
		imageID, productID).Scan(&wasPrimary)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("deleting product image: %w", err)
	}

	if wasPrimary {
		// Promote the next one in gallery order. No-op when the product has
		// no images left, which is a legitimate state.
		if _, err := tx.Exec(ctx, `
			UPDATE product_images SET is_primary = TRUE
			WHERE id = (
			    SELECT id FROM product_images
			    WHERE product_id = $1
			    ORDER BY sort_order, id
			    LIMIT 1
			)`, productID); err != nil {
			return fmt.Errorf("promoting new primary image: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing delete-image: %w", err)
	}
	return nil
}

// SaveProductEditorial replaces the highlights and usage cards for every
// locale the caller supplies. Locales absent from the map are left alone, so
// editing the Armenian tab cannot wipe the English one.
func (s *Store) SaveProductEditorial(ctx context.Context, productID int64, byLocale map[domain.Locale]domain.ProductEditorial) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning save-editorial tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Confirm the product exists before writing children — otherwise an
	// unknown id would simply delete nothing and insert nothing, and the
	// admin would get a cheerful 200 for an edit that went nowhere.
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT true FROM products WHERE id = $1`, productID).Scan(&exists); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("checking product: %w", err)
	}

	for locale, content := range byLocale {
		if _, err := tx.Exec(ctx,
			`DELETE FROM product_highlights WHERE product_id = $1 AND locale = $2`,
			productID, locale); err != nil {
			return fmt.Errorf("clearing %s highlights: %w", locale, err)
		}
		for i, h := range content.Highlights {
			// Position is assigned here rather than trusted from the client:
			// the PK is (product_id, locale, sort_order), so a duplicated
			// sort_order from a buggy form would collide, and a sparse one
			// would leave gaps that read fine but sort oddly after an edit.
			if _, err := tx.Exec(ctx, `
				INSERT INTO product_highlights (product_id, locale, sort_order, text)
				VALUES ($1, $2, $3, $4)`,
				productID, locale, i, h.Text); err != nil {
				return fmt.Errorf("inserting %s highlight: %w", locale, err)
			}
		}

		if _, err := tx.Exec(ctx,
			`DELETE FROM product_usage_cards WHERE product_id = $1 AND locale = $2`,
			productID, locale); err != nil {
			return fmt.Errorf("clearing %s usage cards: %w", locale, err)
		}
		for i, c := range content.UsageCards {
			if _, err := tx.Exec(ctx, `
				INSERT INTO product_usage_cards (product_id, locale, sort_order, kicker, title, body)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				productID, locale, i, c.Kicker, c.Title, c.Body); err != nil {
				return fmt.Errorf("inserting %s usage card: %w", locale, err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing save-editorial: %w", err)
	}
	return nil
}

// SaveProductRelated replaces the curated "Often taken together" list.
//
// Self-references are dropped here rather than left to the CHECK constraint:
// a shop owner ticking the product they are editing is an obvious slip, and
// answering it with a 500-shaped constraint error would be a worse
// experience than quietly doing the sensible thing.
func (s *Store) SaveProductRelated(ctx context.Context, productID int64, relatedIDs []int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning save-related tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`DELETE FROM product_related WHERE product_id = $1`, productID); err != nil {
		return fmt.Errorf("clearing related products: %w", err)
	}

	seen := make(map[int64]bool, len(relatedIDs))
	order := 0
	for _, id := range relatedIDs {
		if id == productID || seen[id] {
			continue
		}
		seen[id] = true
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_related (product_id, related_id, sort_order)
			VALUES ($1, $2, $3)`,
			productID, id, order); err != nil {
			var pgErr interface{ SQLState() string }
			if errors.As(err, &pgErr) && pgErr.SQLState() == foreignKeyViolation {
				return domain.ErrNotFound
			}
			return fmt.Errorf("inserting related product: %w", err)
		}
		order++
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing save-related: %w", err)
	}
	return nil
}
