package store

import (
	"context"
	"errors"
	"fmt"

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
		INSERT INTO products (category_id, slug, name, description, image_url, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		p.CategoryID, p.Slug, p.Name, p.Description, p.ImageURL, p.IsActive,
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
			INSERT INTO product_variants (product_id, sku, label, price_minor, stock_qty)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id`,
			p.ID, v.SKU, v.Label, v.PriceMinor, v.StockQty,
		).Scan(&v.ID)
		if err != nil {
			if mapped := mapProductConstraint(err); mapped != nil {
				return mapped
			}
			return fmt.Errorf("inserting variant %q: %w", v.SKU, err)
		}
		v.ProductID = p.ID
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing create-product: %w", err)
	}
	return nil
}

// UpdateProduct updates the mutable fields (slug stays immutable — it is a
// public URL).
func (s *Store) UpdateProduct(ctx context.Context, p *domain.Product) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE products
		SET category_id = $1, name = $2, description = $3, image_url = $4, is_active = $5
		WHERE id = $6`,
		p.CategoryID, p.Name, p.Description, p.ImageURL, p.IsActive, p.ID)
	if err != nil {
		if mapped := mapProductConstraint(err); mapped != nil {
			return mapped
		}
		return fmt.Errorf("updating product: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) UpdateProductImage(ctx context.Context, productID int64, imageURL string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE products SET image_url = $1 WHERE id = $2`, imageURL, productID)
	if err != nil {
		return fmt.Errorf("updating product image: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (s *Store) UpdateVariant(ctx context.Context, variantID, priceMinor int64, stockQty int) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE product_variants SET price_minor = $1, stock_qty = $2 WHERE id = $3`,
		priceMinor, stockQty, variantID)
	if err != nil {
		return fmt.Errorf("updating variant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
