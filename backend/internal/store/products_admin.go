package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// mapProductConstraint translates a Postgres unique/FK violation into the
// matching domain sentinel using the CONSTRAINT NAME — one DB error code
// (23505) can mean different business problems.
func mapProductConstraint(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}
	switch pgErr.ConstraintName {
	case "products_slug_key":
		return domain.ErrSlugTaken
	case "product_variants_sku_key":
		return domain.ErrSKUTaken
	case "product_variants_product_id_label_key":
		return domain.ErrVariantLabelTaken
	case "products_category_id_fkey":
		return domain.ErrCategoryNotFound
	}
	return nil
}

// CreateProduct inserts a product WITH all its variants in one transaction:
// either the full product appears, or nothing does.
func (s *Store) CreateProduct(ctx context.Context, p *domain.Product) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning create-product tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	err = tx.QueryRow(ctx, `
		INSERT INTO products (category_id, slug, name, description, is_active,
		                      lab_batch, is_cold_chain)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		p.CategoryID, p.Slug, p.Name, p.Description, p.IsActive,
		p.LabBatch, p.IsColdChain,
	).Scan(&p.ID, &p.CreatedAt)
	if err != nil {
		if mapped := mapProductConstraint(err); mapped != nil {
			return mapped
		}
		return fmt.Errorf("inserting product: %w", err)
	}

	for i := range p.Variants {
		v := &p.Variants[i]
		err = tx.QueryRow(ctx, `
			INSERT INTO product_variants (product_id, sku, label, stock_qty)
			VALUES ($1, $2, $3, $4)
			RETURNING id`,
			p.ID, v.SKU, v.Label, v.StockQty,
		).Scan(&v.ID)
		if err != nil {
			if mapped := mapProductConstraint(err); mapped != nil {
				return mapped
			}
			return fmt.Errorf("inserting variant %q: %w", v.SKU, err)
		}
		v.ProductID = p.ID
		if err := upsertVariantPrices(ctx, tx, v.ID, v.Prices); err != nil {
			return err
		}
	}

	if err := upsertProductTranslations(ctx, tx, p); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing create-product: %w", err)
	}
	return nil
}

// upsertProductTranslations writes the English row from p.Name/p.Description
// and one row per additional locale.
//
// ON CONFLICT DO UPDATE rather than DELETE-then-INSERT: the composite primary
// key (product_id, locale) already identifies the row, so an upsert makes the
// operation idempotent without a window where the product has no text at all.
// It also means create and update share this one function.
func upsertProductTranslations(ctx context.Context, tx pgx.Tx, p *domain.Product) error {
	texts := map[domain.Locale]domain.ProductText{
		domain.DefaultLocale: {
			Name: p.Name, Description: p.Description,
			Disclaimer: p.Disclaimer, StorageNote: p.StorageNote,
			HarvestNote: p.HarvestNote, ShippingNote: p.ShippingNote,
		},
	}
	for locale, text := range p.Translations {
		texts[locale] = text
	}

	for locale, text := range texts {
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_translations
			    (product_id, locale, name, description,
			     disclaimer, storage_note, harvest_note, shipping_note)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (product_id, locale)
			DO UPDATE SET name          = EXCLUDED.name,
			              description   = EXCLUDED.description,
			              disclaimer    = EXCLUDED.disclaimer,
			              storage_note  = EXCLUDED.storage_note,
			              harvest_note  = EXCLUDED.harvest_note,
			              shipping_note = EXCLUDED.shipping_note`,
			p.ID, locale, text.Name, text.Description,
			text.Disclaimer, text.StorageNote, text.HarvestNote, text.ShippingNote,
		); err != nil {
			return fmt.Errorf("upserting %s product translation: %w", locale, err)
		}
	}
	return nil
}

// UpdateProduct updates the mutable fields (slug stays immutable — it is a
// public URL) and rewrites the product's translations.
//
// Transactional since E1.5: the product row and its translation rows are one
// edit, and committing the first without the second would leave a product
// whose Armenian name still describes the previous product.
func (s *Store) UpdateProduct(ctx context.Context, p *domain.Product) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning update-product tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE products
		SET category_id = $1, name = $2, description = $3, is_active = $4,
		    lab_batch = $5, is_cold_chain = $6
		WHERE id = $7`,
		p.CategoryID, p.Name, p.Description, p.IsActive,
		p.LabBatch, p.IsColdChain, p.ID)
	if err != nil {
		if mapped := mapProductConstraint(err); mapped != nil {
			return mapped
		}
		return fmt.Errorf("updating product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	if err := upsertProductTranslations(ctx, tx, p); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing update-product: %w", err)
	}
	return nil
}

// UpdateVariant sets stock and the whole price set at once.
//
// The prices are replaced WHOLESALE — every currency the caller sent is
// written, every currency it omitted is deleted. A per-currency PATCH would
// be friendlier to a script and much worse for a form: the admin's price
// editor sends what the boxes currently say, and if omitting AMD meant
// "leave it alone" there would be no way to *remove* an AMD price and let it
// fall back to conversion again. "The request describes the desired state"
// is the same PUT-shaped promise the cart's set-quantity endpoint makes.
func (s *Store) UpdateVariant(ctx context.Context, variantID int64, prices domain.Money, stockQty int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning update-variant tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx,
		`UPDATE product_variants SET stock_qty = $1 WHERE id = $2`, stockQty, variantID)
	if err != nil {
		return fmt.Errorf("updating variant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	if err := upsertVariantPrices(ctx, tx, variantID, prices); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing update-variant: %w", err)
	}
	return nil
}

// upsertVariantPrices makes variant_prices match `prices` exactly.
//
// Upsert-then-delete rather than delete-then-insert, so the variant is never
// momentarily unpriced — the same reasoning as upsertProductTranslations,
// and it matters more here because an empty window would make the product
// briefly unbuyable rather than briefly untranslated.
func upsertVariantPrices(ctx context.Context, tx pgx.Tx, variantID int64, prices domain.Money) error {
	kept := make([]string, 0, len(prices))
	for currency, minor := range prices {
		if _, err := tx.Exec(ctx, `
			INSERT INTO variant_prices (variant_id, currency, price_minor)
			VALUES ($1, $2, $3)
			ON CONFLICT (variant_id, currency)
			DO UPDATE SET price_minor = EXCLUDED.price_minor`,
			variantID, currency, minor); err != nil {
			return fmt.Errorf("upserting %s price for variant %d: %w", currency, variantID, err)
		}
		kept = append(kept, string(currency))
	}

	// Removing a shelf price is a real edit: it puts the variant back on the
	// converted fallback rather than leaving a stale figure behind.
	if _, err := tx.Exec(ctx,
		`DELETE FROM variant_prices WHERE variant_id = $1 AND currency <> ALL($2::text[])`,
		variantID, kept); err != nil {
		return fmt.Errorf("pruning prices for variant %d: %w", variantID, err)
	}
	return nil
}
