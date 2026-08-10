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

// ── The catalog query, in pieces ──────────────────────────────────────────
//
// The list, the count and the facets are three queries over ONE definition
// of "which products match". Era I kept the list and the count as two
// separate literals with a comment warning that their WHERE clauses must
// stay identical; E2 adds a third reader and three more filters, at which
// point a comment is not a good enough guarantee. So the shared parts are
// Go constants that every query concatenates.
//
// Building SQL by concatenation is exactly what "never build SQL by
// concatenation" warns about — but the rule is about USER INPUT, not about
// the `+` operator. Every fragment below is a compile-time constant with no
// runtime value in it (Go computes the final string before main runs, much
// like a constexpr), and every user-supplied value still arrives as a bound
// parameter. Nothing a visitor types can change the shape of the query.
//
// The parameters are numbered once, here, and mean the same thing in all
// three queries:
//
//	$1 category slug ('' = all)   $6  locale
//	$2 include inactive           $7  text search configuration
//	$3 search text ('' = none)    $8  benefit slugs (text[], empty = all)
//	$4 search text, cleaned       $9  price floor, minor units (NULL = none)
//	$5 $4 with LIKE escaped       $10 price ceiling, minor units
//
// ...and the list query adds $11 LIMIT / $12 OFFSET.
const (
	// The three-level fallback: the requested locale, then English, then the
	// pre-translation columns on products itself. The last level exists
	// because a product created through the admin has no translation rows
	// yet — see the note in migration 000007.
	sqlProductName = `COALESCE(t.name, en.name, p.name)`
	sqlProductDesc = `COALESCE(t.description, en.description, p.description)`
	sqlProductTSV  = `COALESCE(t.search_tsv, en.search_tsv, p.search_tsv)`
	// The card's eyebrow, resolved the same way one level up the join.
	sqlCategoryName = `COALESCE(ct.name, cen.name, c.name)`
)

// productSource is the FROM block. The LATERAL subquery is the interesting
// part: for each product row it runs a small aggregate over that product's
// variants, and its results are usable in the WHERE and ORDER BY of the
// outer query — which a plain scalar subquery in the SELECT list could not
// be. One pass over the variants answers three questions at once:
//
//	min_price  the "from" price the card shows, and the price sort key
//	max_price  the top of the product's range, for the slider bounds
//	in_band    how many variants fall inside the requested price window
//
// count(*) FILTER (WHERE …) is the aggregate FILTER clause: a per-aggregate
// WHERE, so several differently-filtered aggregates can share one scan.
// Without it, in_band would need its own correlated subquery.
const productSource = `
	FROM products p
	JOIN categories c ON c.id = p.category_id
	LEFT JOIN product_translations t  ON t.product_id  = p.id AND t.locale  = $6
	LEFT JOIN product_translations en ON en.product_id = p.id AND en.locale = 'en'
	LEFT JOIN category_translations ct  ON ct.category_id  = c.id AND ct.locale  = $6
	LEFT JOIN category_translations cen ON cen.category_id = c.id AND cen.locale = 'en'
	LEFT JOIN LATERAL (
	    SELECT min(v.price_minor) AS min_price,
	           max(v.price_minor) AS max_price,
	           count(*) FILTER (
	               WHERE ($9::bigint IS NULL OR v.price_minor >= $9)
	                 AND ($10::bigint IS NULL OR v.price_minor <= $10)
	           ) AS in_band
	    FROM product_variants v
	    WHERE v.product_id = p.id
	) pr ON TRUE`

// The four predicates, kept apart because the facet query needs to apply
// them SELECTIVELY — a facet's own filter must not narrow its own counts,
// or clicking "Honey" would leave the sidebar reading "Honey 1" and every
// other category 0, and there would be no way back.
const (
	// Always applied: visibility and text search.
	sqlMatchBase = `(p.is_active OR $2)
	  AND ($3 = ''
	       OR ` + sqlProductTSV + ` @@ websearch_to_tsquery($7::regconfig, $3)
	       OR ($4 <> '' AND (` + sqlProductName + ` ILIKE '%' || $5 || '%'
	                         OR word_similarity($4, ` + sqlProductName + `) > 0.35)))`

	sqlMatchCategory = `($1 = '' OR c.slug = $1)`

	// EXISTS rather than a JOIN onto product_benefits: a join would return
	// one row per matching benefit and silently duplicate a product that is
	// good for two of the selected things. EXISTS asks a yes/no question and
	// stops at the first hit.
	sqlMatchBenefit = `(cardinality($8::text[]) = 0 OR EXISTS (
	      SELECT 1 FROM product_benefits pb
	      JOIN benefits b ON b.id = pb.benefit_id
	      WHERE pb.product_id = p.id AND b.slug = ANY($8::text[])))`

	// The first half is not redundant: with no price filter in_band counts
	// every variant, so a product with NO variants would score 0 and vanish
	// from an unfiltered listing. Era I listed such products; this keeps it
	// that way.
	sqlMatchPrice = `(($9::bigint IS NULL AND $10::bigint IS NULL) OR pr.in_band > 0)`

	// What the list and count queries use: all four, ANDed.
	sqlMatchAll = sqlMatchBase +
		` AND ` + sqlMatchCategory +
		` AND ` + sqlMatchBenefit +
		` AND ` + sqlMatchPrice
)

// productOrderBy maps a whitelisted sort onto a constant ORDER BY. Every
// return value is a literal — see the comment on domain.ProductSort for why
// this cannot be a bound parameter.
//
// Each ordering ends with display_name so the result is TOTAL: six products
// that have never sold all have sales_count = 0, and a database is free to
// return equal rows in any order it likes. Without the tiebreak, page 2
// could repeat a product from page 1 — the classic unstable-pagination bug,
// and one that only shows up under load, when the plan changes.
func productOrderBy(f domain.ProductFilter) string {
	var by string
	switch f.EffectiveSort() {
	case domain.SortPriceAsc:
		by = `pr.min_price ASC NULLS LAST, display_name`
	case domain.SortPriceDesc:
		by = `pr.min_price DESC NULLS LAST, display_name`
	case domain.SortNewest:
		by = `p.created_at DESC, display_name`
	default: // domain.SortPopular
		by = `p.sales_count DESC, display_name`
	}

	// Searching with the default sort means "best match first" — a shopper
	// who typed a word wants relevance, not the shop's bestsellers. Choosing
	// a sort explicitly overrides that, which is why this only fires for the
	// default. Ranking sums full-text rank and name similarity, so exact
	// words float above fuzzy ones.
	if f.Search != "" && f.EffectiveSort() == domain.DefaultProductSort {
		return `ts_rank(` + sqlProductTSV + `, websearch_to_tsquery($7::regconfig, $3))
		        + word_similarity($4, ` + sqlProductName + `) DESC, ` + by
	}
	return by
}

// queryArgs packs the ten shared parameters in the documented order.
func queryArgs(f domain.ProductFilter) []any {
	fuzzy := fuzzyQuery(f.Search)
	locale := f.EffectiveLocale()

	// A nil slice reaches Postgres as NULL, and cardinality(NULL) is NULL,
	// not 0 — which would make the benefit predicate neither true nor false
	// and drop every row. An empty non-nil slice encodes as '{}'.
	benefits := f.BenefitSlugs
	if benefits == nil {
		benefits = []string{}
	}

	return []any{
		f.CategorySlug, f.IncludeInactive, f.Search, fuzzy, escapeLike(fuzzy),
		locale, locale.SearchConfig(), benefits, f.PriceMinMinor, f.PriceMaxMinor,
	}
}

// ListProducts returns one page of products (with their variants and
// benefits) and the total count of products matching the filter.
//
// Search matches through THREE doors, widest reach to narrowest:
//  1. full-text ($3, raw): whole words, stemmed, websearch syntax
//  2. substring on name ($5, cleaned+LIKE-escaped): "hon" finds Honey
//  3. fuzzy on name ($4, cleaned): trigram similarity — "hony" → Honey
//
// One imperfection worth naming: a product with no Armenian translation
// matches against an ENGLISH-stemmed tsvector, so an Armenian full-text
// query will not match it well. The trigram doors are language-agnostic and
// still find it, so it degrades rather than disappearing.
func (s *Store) ListProducts(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int, error) {
	listQ := `
		SELECT p.id, p.category_id, p.slug,
		       ` + sqlProductName + ` AS display_name,
		       ` + sqlProductDesc + `,
		       p.image_url, p.is_active, p.created_at,
		       COALESCE(p.badge, ''), p.badge_tone, p.sales_count,
		       c.slug, ` + sqlCategoryName + `
		` + productSource + `
		WHERE ` + sqlMatchAll + `
		ORDER BY ` + productOrderBy(f) + `
		LIMIT $11 OFFSET $12`

	args := queryArgs(f)
	rows, err := s.pool.Query(ctx, listQ, append(args, f.PerPage, f.Offset())...)
	if err != nil {
		return nil, 0, fmt.Errorf("querying products: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.CategoryID, &p.Slug, &p.Name, &p.Description,
			&p.ImageURL, &p.IsActive, &p.CreatedAt,
			&p.Badge, &p.BadgeTone, &p.SalesCount,
			&p.CategorySlug, &p.CategoryName); err != nil {
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
	if err := s.attachBenefits(ctx, products, f.EffectiveLocale()); err != nil {
		return nil, 0, err
	}

	// Same predicate as the page, by construction rather than by comment.
	countQ := `SELECT count(*) ` + productSource + ` WHERE ` + sqlMatchAll
	var total int
	if err := s.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
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
		       `+sqlProductName+`,
		       `+sqlProductDesc+`,
		       p.image_url, p.is_active, p.created_at,
		       COALESCE(p.badge, ''), p.badge_tone, p.sales_count,
		       c.slug, `+sqlCategoryName+`
		FROM products p
		JOIN categories c ON c.id = p.category_id
		LEFT JOIN product_translations t  ON t.product_id  = p.id AND t.locale  = $2
		LEFT JOIN product_translations en ON en.product_id = p.id AND en.locale = 'en'
		LEFT JOIN category_translations ct  ON ct.category_id  = c.id AND ct.locale  = $2
		LEFT JOIN category_translations cen ON cen.category_id = c.id AND cen.locale = 'en'
		WHERE p.slug = $1 AND p.is_active`,
		slug, locale,
	).Scan(&p.ID, &p.CategoryID, &p.Slug, &p.Name, &p.Description,
		&p.ImageURL, &p.IsActive, &p.CreatedAt,
		&p.Badge, &p.BadgeTone, &p.SalesCount,
		&p.CategorySlug, &p.CategoryName)
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
	if err := s.attachBenefits(ctx, products, locale); err != nil {
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

// attachBenefits is attachVariants for the many-to-many side: same ANY($1)
// batching, one extra hop through the join table.
//
// The name falls back locale → English → slug. The third level differs from
// products: a benefit has no pre-translation column to fall back to, so the
// last resort is its slug — ugly on screen, but a chip reading "energy" is
// a better failure than a chip reading nothing.
func (s *Store) attachBenefits(ctx context.Context, products []domain.Product, locale domain.Locale) error {
	if len(products) == 0 {
		return nil
	}

	ids := make([]int64, len(products))
	byID := make(map[int64]*domain.Product, len(products))
	for i := range products {
		ids[i] = products[i].ID
		byID[products[i].ID] = &products[i]
		products[i].Benefits = make([]domain.Benefit, 0)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT pb.product_id, b.id, b.slug, b.sort_order,
		       COALESCE(bt.name, ben.name, b.slug)
		FROM product_benefits pb
		JOIN benefits b ON b.id = pb.benefit_id
		LEFT JOIN benefit_translations bt  ON bt.benefit_id  = b.id AND bt.locale  = $2
		LEFT JOIN benefit_translations ben ON ben.benefit_id = b.id AND ben.locale = 'en'
		WHERE pb.product_id = ANY($1)
		ORDER BY b.sort_order, b.id`,
		ids, locale)
	if err != nil {
		return fmt.Errorf("querying product benefits: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var productID int64
		var b domain.Benefit
		if err := rows.Scan(&productID, &b.ID, &b.Slug, &b.SortOrder, &b.Name); err != nil {
			return fmt.Errorf("scanning product benefit row: %w", err)
		}
		p := byID[productID]
		p.Benefits = append(p.Benefits, b)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating product benefit rows: %w", err)
	}
	return nil
}

// CatalogFacets answers the whole Shop sidebar in one round trip.
//
// The rule of faceted search: a facet group's own filter does not narrow its
// own counts. Picking "Honey" must leave "Beeswax 1" visible, or the only
// way out of a filter is the browser's back button. So the three groups see
// three different result sets:
//
//	category counts   base ∧ benefit ∧ price   (category filter lifted)
//	benefit counts    base ∧ category ∧ price  (benefit filter lifted)
//	price bounds      base ∧ category ∧ benefit
//
// The `base` CTE computes the shared half once and tags each row with the
// three predicates as BOOLEAN COLUMNS rather than applying them. Postgres
// materialises a CTE that is referenced more than once, so the products are
// scanned once and each aggregate re-reads the same small result — the
// alternative being three round trips or three separate scans.
//
// The three groups have different shapes, so they are UNIONed into one
// tagged row format (kind, slug, name, n, min, max, sort_order) and sorted
// back apart in Go. A discriminated union, in the C++ sense: the `kind`
// column says which fields are meaningful.
func (s *Store) CatalogFacets(ctx context.Context, f domain.ProductFilter) (domain.CatalogFacets, error) {
	const q = `
		WITH base AS (
		    SELECT p.id, p.category_id, pr.min_price, pr.max_price,
		           ` + sqlMatchCategory + ` AS ok_category,
		           ` + sqlMatchBenefit + ` AS ok_benefit,
		           ` + sqlMatchPrice + ` AS ok_price
		    ` + productSource + `
		    WHERE ` + sqlMatchBase + `
		)
		SELECT 'category' AS kind, c.slug, COALESCE(ct.name, cen.name, c.name) AS name,
		       count(base.id) FILTER (WHERE base.ok_benefit AND base.ok_price)::int AS n,
		       NULL::bigint AS min_price, NULL::bigint AS max_price,
		       c.sort_order
		FROM categories c
		-- LEFT JOIN, so a category no product matches still comes back with
		-- a 0 instead of vanishing. count(base.id) rather than count(*) for
		-- the same reason: on an unmatched row base.id is NULL, and count of
		-- a column ignores NULLs, whereas count(*) counts the row itself and
		-- would report 1 for every empty category.
		LEFT JOIN base ON base.category_id = c.id
		LEFT JOIN category_translations ct  ON ct.category_id  = c.id AND ct.locale  = $6
		LEFT JOIN category_translations cen ON cen.category_id = c.id AND cen.locale = 'en'
		GROUP BY c.id, c.slug, c.name, ct.name, cen.name, c.sort_order
		-- A zero count is worth showing when a FILTER caused it: "Beeswax 0"
		-- tells you the click you are about to make returns nothing, and
		-- keeps the sidebar from reshuffling under the cursor. A category
		-- with nothing in it at all is different — it is noise. This
		-- HAVING counts base rows WITHOUT the ok_* filters, so the row
		-- survives only if the category holds something visible.
		--
		-- Not hypothetical: this dev database still has Era I's herbal-tea
		-- and coffee categories, kept alive by deactivated products that
		-- old orders reference and so cannot be deleted.
		HAVING count(base.id) > 0

		UNION ALL

		SELECT 'benefit', b.slug, COALESCE(bt.name, ben.name, b.slug),
		       count(base.id) FILTER (WHERE base.ok_category AND base.ok_price)::int,
		       NULL::bigint, NULL::bigint,
		       b.sort_order
		FROM benefits b
		LEFT JOIN product_benefits pb ON pb.benefit_id = b.id
		LEFT JOIN base ON base.id = pb.product_id
		LEFT JOIN benefit_translations bt  ON bt.benefit_id  = b.id AND bt.locale  = $6
		LEFT JOIN benefit_translations ben ON ben.benefit_id = b.id AND ben.locale = 'en'
		GROUP BY b.id, b.slug, bt.name, ben.name, b.sort_order
		HAVING count(base.id) > 0

		UNION ALL

		-- One summary row: the "All hive products" total (category filter
		-- lifted, like the category counts) and the slider's ends.
		SELECT 'summary', NULL::text, NULL::text,
		       count(*) FILTER (WHERE ok_benefit AND ok_price)::int,
		       min(min_price) FILTER (WHERE ok_category AND ok_benefit),
		       max(max_price) FILTER (WHERE ok_category AND ok_benefit),
		       0
		FROM base

		ORDER BY kind, sort_order, slug`

	rows, err := s.pool.Query(ctx, q, queryArgs(f)...)
	if err != nil {
		return domain.CatalogFacets{}, fmt.Errorf("querying catalog facets: %w", err)
	}
	defer rows.Close()

	facets := domain.CatalogFacets{
		Categories: make([]domain.FacetCount, 0),
		Benefits:   make([]domain.FacetCount, 0),
	}
	for rows.Next() {
		var kind string
		// Pointers because the summary row carries no slug or name, and the
		// facet rows carry no bounds: a nullable column needs a Go type that
		// has a null, which for a scan target means a pointer.
		var slug, name *string
		var n int
		var minPrice, maxPrice *int64
		var sortOrder int
		if err := rows.Scan(&kind, &slug, &name, &n, &minPrice, &maxPrice, &sortOrder); err != nil {
			return domain.CatalogFacets{}, fmt.Errorf("scanning facet row: %w", err)
		}

		switch kind {
		case "category":
			facets.Categories = append(facets.Categories, domain.FacetCount{
				Slug: derefString(slug), Name: derefString(name), Count: n,
			})
		case "benefit":
			facets.Benefits = append(facets.Benefits, domain.FacetCount{
				Slug: derefString(slug), Name: derefString(name), Count: n,
			})
		case "summary":
			facets.Total = n
			// NULL bounds mean "nothing matched" — min() over no rows is
			// NULL, not 0. Left at zero, which the client reads as an empty
			// range and disables the slider for.
			if minPrice != nil {
				facets.PriceMinMinor = *minPrice
			}
			if maxPrice != nil {
				facets.PriceMaxMinor = *maxPrice
			}
		}
	}
	if err := rows.Err(); err != nil {
		return domain.CatalogFacets{}, fmt.Errorf("iterating facet rows: %w", err)
	}
	return facets, nil
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
