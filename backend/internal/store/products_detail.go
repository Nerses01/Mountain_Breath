package store

import (
	"context"
	"fmt"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// Everything the PRODUCT PAGE needs and the shop grid does not.
//
// Kept out of the listing on purpose: a card shows no gallery, no bullets and
// no usage cards, so loading them for twelve products to render twelve cards
// is work nobody sees. The split is the whole reason `GET /products` and
// `GET /products/{slug}` return different shapes.

// attachImages loads the gallery (photos hero first) and the video slot.
//
// ORDER BY is_primary DESC first, then sort_order: the partial unique index
// in 000011 guarantees at most one primary, so this puts the hero at index 0
// without the caller having to search for it.
//
// One query for both kinds, split here in Go: the video is a product_images
// row (migration 000026), but the two answer different rendering questions —
// Images feeds <img> slots, Video a <video> tab — so the domain model keeps
// them apart even though storage does not.
func (s *Store) attachImages(ctx context.Context, p *domain.Product, locale domain.Locale) error {
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.url, i.sort_order, i.is_primary, i.kind,
		       COALESCE(t.alt, en.alt, '')
		FROM product_images i
		LEFT JOIN product_image_translations t  ON t.image_id  = i.id AND t.locale  = $2
		LEFT JOIN product_image_translations en ON en.image_id = i.id AND en.locale = 'en'
		WHERE i.product_id = $1
		ORDER BY i.is_primary DESC, i.sort_order, i.id`,
		p.ID, locale)
	if err != nil {
		return fmt.Errorf("querying product images: %w", err)
	}
	defer rows.Close()

	p.Images = make([]domain.ProductImage, 0)
	for rows.Next() {
		var img domain.ProductImage
		var kind string
		if err := rows.Scan(&img.ID, &img.URL, &img.SortOrder, &img.IsPrimary, &kind, &img.Alt); err != nil {
			return fmt.Errorf("scanning product image: %w", err)
		}
		if kind == "video" {
			video := img
			p.Video = &video
			continue
		}
		p.Images = append(p.Images, img)
	}
	return rows.Err()
}

// attachHighlights loads the "What it does" bullets for ONE locale.
//
// No COALESCE fallback chain here, unlike every other translated read, and
// the difference follows from decision #4: these rows are keyed by locale, so
// "the Armenian bullets" and "the English bullets" are separate LISTS, not
// separate fields of one row. Falling back per bullet would interleave two
// languages inside one panel. Falling back per LIST is the right unit, which
// is what the second query does.
func (s *Store) attachHighlights(ctx context.Context, p *domain.Product, locale domain.Locale) error {
	load := func(l domain.Locale) ([]domain.ProductHighlight, error) {
		rows, err := s.pool.Query(ctx, `
			SELECT sort_order, text
			FROM product_highlights
			WHERE product_id = $1 AND locale = $2
			ORDER BY sort_order`,
			p.ID, l)
		if err != nil {
			return nil, fmt.Errorf("querying highlights: %w", err)
		}
		defer rows.Close()

		out := make([]domain.ProductHighlight, 0)
		for rows.Next() {
			var h domain.ProductHighlight
			if err := rows.Scan(&h.SortOrder, &h.Text); err != nil {
				return nil, fmt.Errorf("scanning highlight: %w", err)
			}
			out = append(out, h)
		}
		return out, rows.Err()
	}

	highlights, err := load(locale)
	if err != nil {
		return err
	}
	// An untranslated product shows the English panel rather than an empty
	// one — the same "degrade, don't disappear" rule the name fallback
	// follows, applied one level up.
	if len(highlights) == 0 && locale != domain.DefaultLocale {
		if highlights, err = load(domain.DefaultLocale); err != nil {
			return err
		}
	}
	p.Highlights = highlights
	return nil
}

// attachUsageCards is attachHighlights for the Morning / Course / Pairs-with
// cards, and falls back as a whole SET for the same reason.
func (s *Store) attachUsageCards(ctx context.Context, p *domain.Product, locale domain.Locale) error {
	load := func(l domain.Locale) ([]domain.ProductUsageCard, error) {
		rows, err := s.pool.Query(ctx, `
			SELECT sort_order, kicker, title, body
			FROM product_usage_cards
			WHERE product_id = $1 AND locale = $2
			ORDER BY sort_order`,
			p.ID, l)
		if err != nil {
			return nil, fmt.Errorf("querying usage cards: %w", err)
		}
		defer rows.Close()

		out := make([]domain.ProductUsageCard, 0)
		for rows.Next() {
			var c domain.ProductUsageCard
			if err := rows.Scan(&c.SortOrder, &c.Kicker, &c.Title, &c.Body); err != nil {
				return nil, fmt.Errorf("scanning usage card: %w", err)
			}
			out = append(out, c)
		}
		return out, rows.Err()
	}

	cards, err := load(locale)
	if err != nil {
		return err
	}
	if len(cards) == 0 && locale != domain.DefaultLocale {
		if cards, err = load(domain.DefaultLocale); err != nil {
			return err
		}
	}
	p.UsageCards = cards
	return nil
}

// relatedLimit is how many products "Often taken together" shows — the
// design draws four.
const relatedLimit = 4

// ListRelated answers "Often taken together" for one product.
//
// Curated first: whatever the admin put in product_related, in their order.
//
// THE FALLBACK IS NOT WHAT THE PLAN SPECIFIED, and the reason is worth
// recording. The plan said "same category by popularity", which is the
// standard rule and is DEAD ON ARRIVAL for this catalog: E2 gave the shop six
// products in six categories, one each, so "another product in this category"
// matches nothing and the panel would always be empty.
//
// So the fallback ranks every other active product by how many BENEFITS it
// shares with this one, then by popularity. For an apiary that is also the
// better signal — "often taken together" is a claim about what the things do,
// not about which shelf they sit on. Products sharing nothing still appear,
// last, so the panel is never empty while the shop has other products.
func (s *Store) ListRelated(ctx context.Context, slug string, view domain.View) ([]domain.Product, error) {
	curated, err := s.ListCuratedRelated(ctx, slug, view)
	if err != nil {
		return nil, err
	}
	if len(curated) > 0 {
		return curated, nil
	}

	return s.relatedProducts(ctx, slug, view, `
		JOIN products src ON src.slug = $1
		LEFT JOIN LATERAL (
		    -- How many of this product's benefits the source product also
		    -- has. A LATERAL because the count is per candidate row, and a
		    -- correlated subquery in ORDER BY could not be reused in SELECT.
		    SELECT count(*) AS shared
		    FROM product_benefits a
		    JOIN product_benefits b ON b.benefit_id = a.benefit_id
		    WHERE a.product_id = p.id AND b.product_id = src.id
		) rel ON TRUE
		WHERE p.is_active AND p.id <> src.id
		ORDER BY rel.shared DESC, p.sales_count DESC, p.id`)
}

// ListCuratedRelated returns ONLY what the admin curated — empty when
// nothing is, where ListRelated would compute a fallback.
//
// The admin editor needs this distinction and the storefront does not: a
// picker pre-filled from ListRelated would show the COMPUTED list as though
// it were curated, and saving it would silently freeze a dynamic panel into
// a static one. Worse, a picker that starts empty for a product that IS
// curated invites an admin to wipe the curation by saving.
func (s *Store) ListCuratedRelated(ctx context.Context, slug string, view domain.View) ([]domain.Product, error) {
	return s.relatedProducts(ctx, slug, view, `
		JOIN product_related pr ON pr.related_id = p.id
		JOIN products src ON src.id = pr.product_id AND src.slug = $1
		WHERE p.is_active
		ORDER BY pr.sort_order, p.id`)
}

// relatedProducts runs one of the two strategies above. The `tail` is a
// compile-time constant supplied by the caller — the same
// constants-not-user-input rule the catalog queries follow.
func (s *Store) relatedProducts(ctx context.Context, slug string, view domain.View, tail string) ([]domain.Product, error) {
	return s.productCards(ctx, view, tail, relatedLimit, slug)
}

// productCards is the one query shape behind every "grid of cards keyed by
// something else" read — related products (keyed by a slug) and E8's
// wishlist (keyed by a user id). The tail supplies the JOIN/WHERE/ORDER and
// refers to the caller's key as $1; $2 is always the locale. Generalized
// out of relatedProducts when the wishlist would otherwise have copied the
// whole SELECT list — the orderColumns lesson, applied preemptively.
func (s *Store) productCards(ctx context.Context, view domain.View, tail string, limit int, arg any) ([]domain.Product, error) {
	locale := view.EffectiveLocale()
	q := `
		SELECT p.id, p.category_id, p.slug,
		       ` + sqlProductName + `,
		       ` + sqlProductDesc + `,
		       p.is_active, p.created_at,
		       COALESCE(p.badge, ''), p.badge_tone, p.sales_count,
		       c.slug, ` + sqlCategoryName + `,
		       p.rating_avg::float8, p.rating_count
		FROM products p
		JOIN categories c ON c.id = p.category_id
		LEFT JOIN product_translations t  ON t.product_id  = p.id AND t.locale  = $2
		LEFT JOIN product_translations en ON en.product_id = p.id AND en.locale = 'en'
		LEFT JOIN category_translations ct  ON ct.category_id  = c.id AND ct.locale  = $2
		LEFT JOIN category_translations cen ON cen.category_id = c.id AND cen.locale = 'en'
		` + tail + `
		LIMIT ` + fmt.Sprint(limit)

	rows, err := s.pool.Query(ctx, q, arg, locale)
	if err != nil {
		return nil, fmt.Errorf("querying product cards: %w", err)
	}
	defer rows.Close()

	products := make([]domain.Product, 0, limit)
	for rows.Next() {
		var p domain.Product
		if err := rows.Scan(&p.ID, &p.CategoryID, &p.Slug, &p.Name, &p.Description,
			&p.IsActive, &p.CreatedAt,
			&p.Badge, &p.BadgeTone, &p.SalesCount,
			&p.CategorySlug, &p.CategoryName,
			&p.Rating.Average, &p.Rating.Count); err != nil {
			return nil, fmt.Errorf("scanning product card: %w", err)
		}
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating product cards: %w", err)
	}

	// The card needs a price, a benefit line and its photos, exactly like
	// the shop grid — this shared tail is what keeps the wishlist's and the
	// related panel's cards from drifting from it.
	if err := s.attachVariants(ctx, products, view); err != nil {
		return nil, err
	}
	if err := s.attachBenefits(ctx, products, locale); err != nil {
		return nil, err
	}
	if err := s.attachCardImages(ctx, products, locale); err != nil {
		return nil, err
	}
	return products, nil
}
