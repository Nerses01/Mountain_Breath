package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// escapeLike neutralizes LIKE wildcards in user input: someone searching
// for "100%" means the literal text, not "anything starting with 100".
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// fuzzyQuery strips websearch operators for the trigram doors: in
// "honey -tea" the "-tea" is an instruction (exclude), not text to match —
// leaving it in would let trigrams literally match the word "tea".
func fuzzyQuery(s string) string {
	var words []string
	for _, w := range strings.Fields(s) {
		if strings.HasPrefix(w, "-") || strings.EqualFold(w, "or") {
			continue
		}
		if w = strings.Trim(w, `"`); w != "" {
			words = append(words, w)
		}
	}
	return strings.Join(words, " ")
}

// ListProducts returns one page of active products (with their variants)
// and the total count of products matching the filter.
func (s *Store) ListProducts(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int, error) {
	// ($1 = '' OR c.slug = $1): one query serves both the filtered and the
	// unfiltered case — no string-built SQL. Same trick for $2 (admins see
	// inactive) and $3/$4 (text search).
	//
	// Search matches through THREE doors, widest reach to narrowest:
	//   1. full-text ($3, raw): whole words, stemmed, websearch syntax
	//   2. substring on name ($5, cleaned+LIKE-escaped): "hon" finds Honey
	//   3. fuzzy on name ($4, cleaned): trigram similarity — "hony" → Honey
	// The trigram doors use the CLEANED query (operators stripped) and are
	// closed entirely when cleaning leaves nothing (pure-exclusion query).
	// Ranking sums FTS rank and name similarity: exact words float above
	// fuzzy matches.
	// $8 is the requested locale and $9 its text search configuration, cast
	// to regconfig. The cast is safe from injection twice over: the value is
	// a bound parameter, and Locale.SearchConfig returns one of three
	// constants — Postgres would reject anything else outright.
	//
	// Text resolves through the same three-level fallback as categories, and
	// search matches the resolved tsvector. The tsvector fallback must go
	// all the way down to p.search_tsv, not stop at en.search_tsv: a product
	// created through the admin has no translation rows at all yet, and
	// stopping one level short silently removed such products from every
	// full-text result. A regression test in search_test.go covers it.
	//
	// One imperfection worth naming: a product with no Armenian translation
	// matches against an ENGLISH-stemmed tsvector, so an Armenian full-text
	// query will not match it well. The trigram doors below are
	// language-agnostic and still find it, so it degrades rather than
	// disappearing.
	const listQ = `
		SELECT p.id, p.category_id, p.slug,
		       COALESCE(t.name, en.name, p.name),
		       COALESCE(t.description, en.description, p.description),
		       p.image_url, p.is_active, p.created_at
		FROM products p
		JOIN categories c ON c.id = p.category_id
		LEFT JOIN product_translations t  ON t.product_id  = p.id AND t.locale  = $8
		LEFT JOIN product_translations en ON en.product_id = p.id AND en.locale = 'en'
		WHERE (p.is_active OR $2) AND ($1 = '' OR c.slug = $1)
		  AND ($3 = ''
		       OR COALESCE(t.search_tsv, en.search_tsv, p.search_tsv) @@ websearch_to_tsquery($9::regconfig, $3)
		       OR ($4 <> '' AND (COALESCE(t.name, en.name, p.name) ILIKE '%' || $5 || '%'
		                         OR word_similarity($4, COALESCE(t.name, en.name, p.name)) > 0.35)))
		ORDER BY
		  CASE WHEN $3 = '' THEN NULL
		       ELSE ts_rank(COALESCE(t.search_tsv, en.search_tsv, p.search_tsv), websearch_to_tsquery($9::regconfig, $3))
		            + word_similarity($4, COALESCE(t.name, en.name, p.name))
		  END DESC NULLS LAST,
		  COALESCE(t.name, en.name, p.name)
		LIMIT $6 OFFSET $7`

	fuzzy := fuzzyQuery(f.Search)
	locale := f.EffectiveLocale()
	rows, err := s.pool.Query(ctx, listQ,
		f.CategorySlug, f.IncludeInactive, f.Search, fuzzy, escapeLike(fuzzy),
		f.PerPage, f.Offset(), locale, locale.SearchConfig())
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

	// The WHERE clause must stay identical to listQ's, or the total would
	// describe a different result set than the page.
	var total int
	const countQ = `
		SELECT count(*)
		FROM products p
		JOIN categories c ON c.id = p.category_id
		LEFT JOIN product_translations t  ON t.product_id  = p.id AND t.locale  = $6
		LEFT JOIN product_translations en ON en.product_id = p.id AND en.locale = 'en'
		WHERE (p.is_active OR $2) AND ($1 = '' OR c.slug = $1)
		  AND ($3 = ''
		       OR COALESCE(t.search_tsv, en.search_tsv, p.search_tsv) @@ websearch_to_tsquery($7::regconfig, $3)
		       OR ($4 <> '' AND (COALESCE(t.name, en.name, p.name) ILIKE '%' || $5 || '%'
		                         OR word_similarity($4, COALESCE(t.name, en.name, p.name)) > 0.35)))`
	if err := s.pool.QueryRow(ctx, countQ,
		f.CategorySlug, f.IncludeInactive, f.Search, fuzzy, escapeLike(fuzzy),
		locale, locale.SearchConfig()).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("counting products: %w", err)
	}

	return products, total, nil
}

func (s *Store) GetProductBySlug(ctx context.Context, slug string, locale domain.Locale) (domain.Product, error) {
	// The slug is deliberately NOT translated: it is the product's stable
	// identity in the URL, so /products/wildflower-honey resolves to the same
	// product in every language and a link shared between speakers still
	// works. Only the display text changes.
	var p domain.Product
	err := s.pool.QueryRow(ctx, `
		SELECT p.id, p.category_id, p.slug,
		       COALESCE(t.name, en.name, p.name),
		       COALESCE(t.description, en.description, p.description),
		       p.image_url, p.is_active, p.created_at
		FROM products p
		LEFT JOIN product_translations t  ON t.product_id  = p.id AND t.locale  = $2
		LEFT JOIN product_translations en ON en.product_id = p.id AND en.locale = 'en'
		WHERE p.slug = $1 AND p.is_active`,
		slug, locale,
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
