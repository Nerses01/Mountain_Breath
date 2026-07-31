package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// ListProducts returns one page of active products (with their variants)
// and the total count of products matching the filter.
func (s *Store) ListProducts(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int, error) {
	// ($1 = '' OR c.slug = $1): one query serves both the filtered and the
	// unfiltered case — no string-built SQL. Same trick for $2 (admins see
	// inactive) and $3 (text search). websearch_to_tsquery parses human
	// input safely: quoted phrases, OR, minus-exclusion — never SQL.
	// Ordering: by relevance rank when searching, alphabetical otherwise.
	const listQ = `
		SELECT p.id, p.category_id, p.slug, p.name, p.description, p.image_url, p.is_active, p.created_at
		FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE (p.is_active OR $2) AND ($1 = '' OR c.slug = $1)
		  AND ($3 = '' OR p.search_tsv @@ websearch_to_tsquery('english', $3))
		ORDER BY
		  CASE WHEN $3 = '' THEN NULL
		       ELSE ts_rank(p.search_tsv, websearch_to_tsquery('english', $3))
		  END DESC NULLS LAST,
		  p.name
		LIMIT $4 OFFSET $5`

	rows, err := s.pool.Query(ctx, listQ, f.CategorySlug, f.IncludeInactive, f.Search, f.PerPage, f.Offset())
	if err != nil {
		return nil, 0, fmt.Errorf("querying products: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.CategoryID, &p.Slug, &p.Name, &p.Description,
			&p.ImageURL, &p.IsActive, &p.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scanning product row: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterating product rows: %w", err)
	}

	if err := s.attachVariants(ctx, products); err != nil {
		return nil, 0, err
	}

	var total int
	const countQ = `
		SELECT count(*)
		FROM products p
		JOIN categories c ON c.id = p.category_id
		WHERE (p.is_active OR $2) AND ($1 = '' OR c.slug = $1)
		  AND ($3 = '' OR p.search_tsv @@ websearch_to_tsquery('english', $3))`
	if err := s.pool.QueryRow(ctx, countQ, f.CategorySlug, f.IncludeInactive, f.Search).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting products: %w", err)
	}

	return products, total, nil
}

func (s *Store) GetProductBySlug(ctx context.Context, slug string) (domain.Product, error) {
	var p domain.Product
	err := s.pool.QueryRow(ctx, `
		SELECT id, category_id, slug, name, description, image_url, is_active, created_at
		FROM products
		WHERE slug = $1 AND is_active`,
		slug,
	).Scan(&p.ID, &p.CategoryID, &p.Slug, &p.Name, &p.Description,
		&p.ImageURL, &p.IsActive, &p.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Product{}, domain.ErrNotFound
		}
		return domain.Product{}, fmt.Errorf("querying product %q: %w", slug, err)
	}

	products := []domain.Product{p}
	if err := s.attachVariants(ctx, products); err != nil {
		return domain.Product{}, err
	}
	return products[0], nil
}

// attachVariants loads the variants for all given products in ONE query
// (WHERE product_id = ANY(...)) instead of one query per product — the
// classic N+1 problem, avoided.
func (s *Store) attachVariants(ctx context.Context, products []domain.Product) error {
	if len(products) == 0 {
		return nil
	}

	ids := make([]int64, len(products))
	byID := make(map[int64]*domain.Product, len(products))
	for i := range products {
		ids[i] = products[i].ID
		byID[products[i].ID] = &products[i]
		products[i].Variants = make([]domain.ProductVariant, 0)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, product_id, sku, label, price_minor, stock_qty
		FROM product_variants
		WHERE product_id = ANY($1)
		ORDER BY price_minor`,
		ids)
	if err != nil {
		return fmt.Errorf("querying variants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v domain.ProductVariant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.SKU, &v.Label, &v.PriceMinor, &v.StockQty); err != nil {
			return fmt.Errorf("scanning variant row: %w", err)
		}
		p := byID[v.ProductID]
		p.Variants = append(p.Variants, v)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating variant rows: %w", err)
	}
	return nil
}
