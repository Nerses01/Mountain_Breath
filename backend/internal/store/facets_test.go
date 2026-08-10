package store_test

import (
	"context"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// seedHive builds a miniature version of the design's catalog: four
// products, three categories, and benefits overlapping in both directions
// (one product with two benefits, one benefit on two products). The overlap
// is the point — a fixture where every product had exactly one benefit
// would pass with a plain JOIN and never notice the duplicate rows a
// many-to-many join produces.
//
//	product   category   benefits            prices (minor)
//	honey     honey      energy, sweetening  1400, 2600
//	wax       beeswax    skin                 900
//	jelly     jelly      energy, skin        3200, 5800
//	pollen    pollen     energy              1600
func seedHive(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if _, err := testPool.Exec(ctx, `
		INSERT INTO categories (slug, name, sort_order) VALUES
		    ('honey', 'Honey', 1), ('beeswax', 'Beeswax', 2),
		    ('royal-jelly', 'Royal jelly', 3), ('bee-pollen', 'Bee pollen', 4)`); err != nil {
		t.Fatalf("seeding categories: %v", err)
	}

	if _, err := testPool.Exec(ctx, `
		INSERT INTO products (category_id, slug, name, description, badge, badge_tone, sales_count)
		SELECT c.id, v.slug, v.name, '', v.badge, 'honey', v.sales
		FROM (VALUES
		    ('honey',       'honey',  'Wildflower Honey', 'best_seller', 100),
		    ('beeswax',     'wax',    'Beeswax Blocks',   'for_makers',   10),
		    ('royal-jelly', 'jelly',  'Royal Jelly',      'cold_chain',   50),
		    ('bee-pollen',  'pollen', 'Pollen Granules',  NULL,           70)
		) AS v(cat_slug, slug, name, badge, sales)
		JOIN categories c ON c.slug = v.cat_slug`); err != nil {
		t.Fatalf("seeding products: %v", err)
	}

	if _, err := testPool.Exec(ctx, `
		INSERT INTO product_variants (product_id, sku, label, price_minor, stock_qty)
		SELECT p.id, v.sku, v.label, v.price, 10
		FROM (VALUES
		    ('honey',  'HON-500', '500 g', 1400),
		    ('honey',  'HON-1K',  '1 kg',  2600),
		    ('wax',    'WAX-400', '400 g',  900),
		    ('jelly',  'RJL-25',  '25 g',  3200),
		    ('jelly',  'RJL-50',  '50 g',  5800),
		    ('pollen', 'POL-250', '250 g', 1600)
		) AS v(product_slug, sku, label, price)
		JOIN products p ON p.slug = v.product_slug`); err != nil {
		t.Fatalf("seeding variants: %v", err)
	}

	// The benefit rows themselves come from migration 000008, not from here
	// — resetDB truncates products and categories but not the taxonomy.
	if _, err := testPool.Exec(ctx, `
		INSERT INTO product_benefits (product_id, benefit_id)
		SELECT p.id, b.id
		FROM (VALUES
		    ('honey',  'energy'), ('honey', 'sweetening'),
		    ('wax',    'skin'),
		    ('jelly',  'energy'), ('jelly', 'skin'),
		    ('pollen', 'energy')
		) AS v(product_slug, benefit_slug)
		JOIN products p ON p.slug = v.product_slug
		JOIN benefits b ON b.slug = v.benefit_slug`); err != nil {
		t.Fatalf("seeding product_benefits: %v", err)
	}
}

func slugsOf(products []domain.Product) []string {
	out := make([]string, 0, len(products))
	for _, p := range products {
		out = append(out, p.Slug)
	}
	return out
}

func equalSlugs(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func ptr(n int64) *int64 { return &n }

func TestListProducts_Filters(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	tests := []struct {
		name   string
		filter domain.ProductFilter
		want   []string // sorted by name, the default tiebreak
	}{
		{
			name:   "no filter returns everything",
			filter: domain.ProductFilter{},
			want:   []string{"honey", "wax", "jelly", "pollen"},
		},
		{
			name:   "category",
			filter: domain.ProductFilter{CategorySlug: "honey"},
			want:   []string{"honey"},
		},
		{
			name:   "one benefit",
			filter: domain.ProductFilter{BenefitSlugs: []string{"skin"}},
			want:   []string{"wax", "jelly"},
		},
		{
			// Two chips WIDEN the result (OR), and honey+jelly both carry
			// energy — so a product matching twice must still appear once.
			name:   "two benefits are a union, not a duplicate",
			filter: domain.ProductFilter{BenefitSlugs: []string{"energy", "sweetening"}},
			want:   []string{"honey", "jelly", "pollen"},
		},
		{
			name:   "unknown benefit matches nothing",
			filter: domain.ProductFilter{BenefitSlugs: []string{"telepathy"}},
			want:   nil,
		},
		{
			// Different facets narrow each other: honey is in the honey
			// category AND good for energy.
			name: "category and benefit combine with AND",
			filter: domain.ProductFilter{
				CategorySlug: "honey", BenefitSlugs: []string{"skin"},
			},
			want: nil,
		},
		{
			// A product matches if ANY variant is in the band. Honey's 1 kg
			// is 2600, so honey is in — even though its 500 g is not.
			name:   "price floor",
			filter: domain.ProductFilter{PriceMinMinor: ptr(2000)},
			want:   []string{"honey", "jelly"},
		},
		{
			// Pollen's only variant is 1600, so it is out; honey qualifies on
			// its 500 g even though its 1 kg is over the ceiling.
			name:   "price ceiling",
			filter: domain.ProductFilter{PriceMaxMinor: ptr(1500)},
			want:   []string{"wax", "honey"},
		},
		{
			name: "price band excludes a product whose range straddles it",
			// Jelly is 3200/5800 — nothing between 1500 and 3000, so it is
			// out. A naive "product min <= max AND product max >= min"
			// overlap test would wrongly include it.
			filter: domain.ProductFilter{PriceMinMinor: ptr(1500), PriceMaxMinor: ptr(3000)},
			want:   []string{"honey", "pollen"},
		},
		{
			name: "everything at once",
			filter: domain.ProductFilter{
				BenefitSlugs:  []string{"energy"},
				PriceMinMinor: ptr(1000),
				PriceMaxMinor: ptr(2000),
			},
			want: []string{"honey", "pollen"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.filter
			f.Page, f.PerPage = 1, 20
			f.Sort = domain.SortPriceAsc // stable, independent of sales_count

			got, total, err := s.ListProducts(ctx, f)
			if err != nil {
				t.Fatalf("ListProducts: %v", err)
			}
			if total != len(tc.want) {
				t.Errorf("total = %d, want %d", total, len(tc.want))
			}

			// Compare as a set: the sort is asserted separately below.
			gotSlugs := slugsOf(got)
			if len(gotSlugs) != len(tc.want) {
				t.Fatalf("slugs = %v, want %v", gotSlugs, tc.want)
			}
			for _, w := range tc.want {
				found := false
				for _, g := range gotSlugs {
					if g == w {
						found = true
					}
				}
				if !found {
					t.Errorf("slugs = %v, missing %q", gotSlugs, w)
				}
			}
		})
	}
}

func TestListProducts_Sorts(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	tests := []struct {
		name string
		sort domain.ProductSort
		want []string
	}{
		// sales_count 100 / 70 / 50 / 10
		{"popular", domain.SortPopular, []string{"honey", "pollen", "jelly", "wax"}},
		// cheapest variant: 900 / 1400 / 1600 / 3200
		{"price ascending", domain.SortPriceAsc, []string{"wax", "honey", "pollen", "jelly"}},
		{"price descending", domain.SortPriceDesc, []string{"jelly", "pollen", "honey", "wax"}},
		{
			// Every fixture row is inserted by one statement, so created_at
			// is identical for all four and the tiebreak decides — which is
			// exactly the case that would be non-deterministic without one.
			"newest falls back to name for equal timestamps",
			domain.SortNewest,
			[]string{"wax", "pollen", "jelly", "honey"},
		},
		// Empty sort is not "no order": it is the default.
		{"unset", "", []string{"honey", "pollen", "jelly", "wax"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := s.ListProducts(ctx, domain.ProductFilter{
				Page: 1, PerPage: 20, Sort: tc.sort,
			})
			if err != nil {
				t.Fatalf("ListProducts: %v", err)
			}
			if slugs := slugsOf(got); !equalSlugs(slugs, tc.want...) {
				t.Errorf("order = %v, want %v", slugs, tc.want)
			}
		})
	}
}

// Paging must not repeat or drop a row. With four products all sharing a
// created_at and three sharing nothing else useful, an ORDER BY without a
// total tiebreak is free to return them in a different order per query —
// and the same product could land on both pages.
func TestListProducts_PagingIsStable(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	seen := map[string]bool{}
	for page := 1; page <= 2; page++ {
		got, total, err := s.ListProducts(ctx, domain.ProductFilter{
			Page: page, PerPage: 2, Sort: domain.SortNewest,
		})
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		if total != 4 {
			t.Errorf("total on page %d = %d, want 4", page, total)
		}
		for _, p := range got {
			if seen[p.Slug] {
				t.Errorf("product %q appeared on two pages", p.Slug)
			}
			seen[p.Slug] = true
		}
	}
	if len(seen) != 4 {
		t.Errorf("saw %d distinct products across both pages, want 4", len(seen))
	}
}

func TestListProducts_AttachesBadgeAndBenefits(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	got, _, err := s.ListProducts(ctx, domain.ProductFilter{
		Page: 1, PerPage: 20, CategorySlug: "honey",
	})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d products, want 1", len(got))
	}
	p := got[0]

	if p.Badge != "best_seller" || p.BadgeTone != "honey" {
		t.Errorf("badge = %q/%q, want best_seller/honey", p.Badge, p.BadgeTone)
	}
	// Ordered by the taxonomy's sort_order (energy 1 … sweetening 5), not by
	// insertion — the sidebar and the card must agree on chip order.
	if len(p.Benefits) != 2 ||
		p.Benefits[0].Slug != "energy" || p.Benefits[1].Slug != "sweetening" {
		t.Errorf("benefits = %+v, want energy then sweetening", p.Benefits)
	}
	if p.Benefits[0].Name != "Energy" {
		t.Errorf("benefit name = %q, want the translated name", p.Benefits[0].Name)
	}

	// A product with no badge reads as empty, not as a scan error on NULL.
	pollen, err := s.GetProductBySlug(ctx, "pollen", domain.LocaleEN)
	if err != nil {
		t.Fatalf("GetProductBySlug: %v", err)
	}
	if pollen.Badge != "" {
		t.Errorf("badge = %q, want empty for a product with no badge", pollen.Badge)
	}
	if len(pollen.Benefits) != 1 || pollen.Benefits[0].Slug != "energy" {
		t.Errorf("detail benefits = %+v, want energy", pollen.Benefits)
	}
}

// The card's eyebrow is the category NAME, which has to be resolved and
// translated on the product itself — the client has only category_id
// otherwise, and would have to redo the fallback chain to turn it into text.
func TestListProducts_CarriesResolvedCategory(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	ctx := context.Background()

	if _, err := testPool.Exec(ctx, `
		INSERT INTO category_translations (category_id, locale, name)
		SELECT id, 'ru', 'Мёд' FROM categories WHERE slug = 'honey'`); err != nil {
		t.Fatal(err)
	}

	s := store.New(testPool)
	got, _, err := s.ListProducts(ctx, domain.ProductFilter{
		Page: 1, PerPage: 20, CategorySlug: "honey", Locale: domain.LocaleRU,
	})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d products, want 1", len(got))
	}
	if got[0].CategorySlug != "honey" || got[0].CategoryName != "Мёд" {
		t.Errorf("category = %q/%q, want honey/Мёд", got[0].CategorySlug, got[0].CategoryName)
	}

	// An untranslated category falls back rather than blanking the eyebrow.
	beeswax, _, err := s.ListProducts(ctx, domain.ProductFilter{
		Page: 1, PerPage: 20, CategorySlug: "beeswax", Locale: domain.LocaleRU,
	})
	if err != nil {
		t.Fatal(err)
	}
	if beeswax[0].CategoryName != "Beeswax" {
		t.Errorf("untranslated category = %q, want the English fallback", beeswax[0].CategoryName)
	}

	// Detail reads take the same path.
	detail, err := s.GetProductBySlug(ctx, "honey", domain.LocaleRU)
	if err != nil {
		t.Fatal(err)
	}
	if detail.CategoryName != "Мёд" {
		t.Errorf("detail category = %q, want Мёд", detail.CategoryName)
	}
}

func TestListProducts_BenefitNamesFollowLocale(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	got, _, err := s.ListProducts(ctx, domain.ProductFilter{
		Page: 1, PerPage: 20, CategorySlug: "beeswax", Locale: domain.LocaleRU,
	})
	if err != nil {
		t.Fatalf("ListProducts: %v", err)
	}
	if len(got) != 1 || len(got[0].Benefits) != 1 {
		t.Fatalf("got %d products", len(got))
	}
	if name := got[0].Benefits[0].Name; name != "Кожа" {
		t.Errorf("Russian benefit name = %q, want Кожа", name)
	}
}

// The heart of faceted search: a facet's own filter must not narrow its own
// counts, or the sidebar becomes a one-way door.
func TestCatalogFacets_CountsRespectTheOtherFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	countOf := func(rows []domain.FacetCount, slug string) (int, bool) {
		for _, r := range rows {
			if r.Slug == slug {
				return r.Count, true
			}
		}
		return 0, false
	}

	t.Run("unfiltered", func(t *testing.T) {
		f, err := s.CatalogFacets(ctx, domain.ProductFilter{})
		if err != nil {
			t.Fatalf("CatalogFacets: %v", err)
		}
		if f.Total != 4 {
			t.Errorf("total = %d, want 4", f.Total)
		}
		for slug, want := range map[string]int{
			"honey": 1, "beeswax": 1, "royal-jelly": 1, "bee-pollen": 1,
		} {
			if got, ok := countOf(f.Categories, slug); !ok || got != want {
				t.Errorf("category %s = %d (present=%v), want %d", slug, got, ok, want)
			}
		}
		for slug, want := range map[string]int{
			"energy": 3, "skin": 2, "sweetening": 1,
		} {
			if got, ok := countOf(f.Benefits, slug); !ok || got != want {
				t.Errorf("benefit %s = %d (present=%v), want %d", slug, got, ok, want)
			}
		}
		// Cheapest variant in the catalog to the dearest.
		if f.PriceMinMinor != 900 || f.PriceMaxMinor != 5800 {
			t.Errorf("price bounds = %d..%d, want 900..5800", f.PriceMinMinor, f.PriceMaxMinor)
		}
		// A benefit nothing carries is not in the catalog at all, so it does
		// not belong in the sidebar.
		if _, ok := countOf(f.Benefits, "immunity"); ok {
			t.Error("immunity has no products and should not be listed")
		}
	})

	t.Run("a benefit filter narrows the categories but not the benefits", func(t *testing.T) {
		f, err := s.CatalogFacets(ctx, domain.ProductFilter{BenefitSlugs: []string{"skin"}})
		if err != nil {
			t.Fatalf("CatalogFacets: %v", err)
		}

		// Categories see the benefit filter: only wax and jelly are skin.
		if got, _ := countOf(f.Categories, "beeswax"); got != 1 {
			t.Errorf("beeswax = %d, want 1", got)
		}
		if got, _ := countOf(f.Categories, "honey"); got != 0 {
			t.Errorf("honey = %d, want 0 under a skin filter", got)
		}
		// ...but it must still be LISTED, or there is no way to switch to it.
		if _, ok := countOf(f.Categories, "honey"); !ok {
			t.Error("honey disappeared from the sidebar under a skin filter")
		}

		// Benefits do NOT see their own filter — energy still reads 3, which
		// is what makes clicking it from here meaningful.
		if got, _ := countOf(f.Benefits, "energy"); got != 3 {
			t.Errorf("energy = %d, want 3 (its own facet's filter must not apply)", got)
		}
		if f.Total != 2 {
			t.Errorf("total = %d, want 2", f.Total)
		}
		if f.PriceMinMinor != 900 || f.PriceMaxMinor != 5800 {
			t.Errorf("price bounds = %d..%d, want 900..5800 (wax and jelly)",
				f.PriceMinMinor, f.PriceMaxMinor)
		}
	})

	t.Run("a category filter narrows the benefits but not the categories", func(t *testing.T) {
		f, err := s.CatalogFacets(ctx, domain.ProductFilter{CategorySlug: "honey"})
		if err != nil {
			t.Fatalf("CatalogFacets: %v", err)
		}
		if got, _ := countOf(f.Benefits, "energy"); got != 1 {
			t.Errorf("energy = %d, want 1 under a honey filter", got)
		}
		if got, _ := countOf(f.Benefits, "skin"); got != 0 {
			t.Errorf("skin = %d, want 0 under a honey filter", got)
		}
		// Category counts ignore the category filter, so every category
		// still reads its unfiltered total...
		if got, _ := countOf(f.Categories, "beeswax"); got != 1 {
			t.Errorf("beeswax = %d, want 1 (its own facet's filter must not apply)", got)
		}
		// ...and so does the "All hive products" total.
		if f.Total != 4 {
			t.Errorf("total = %d, want 4 (the All row lifts the category filter)", f.Total)
		}
		// The slider, however, describes only the honey on screen.
		if f.PriceMinMinor != 1400 || f.PriceMaxMinor != 2600 {
			t.Errorf("price bounds = %d..%d, want 1400..2600", f.PriceMinMinor, f.PriceMaxMinor)
		}
	})

	t.Run("a search narrows every group and drops empty rows", func(t *testing.T) {
		f, err := s.CatalogFacets(ctx, domain.ProductFilter{Search: "jelly"})
		if err != nil {
			t.Fatalf("CatalogFacets: %v", err)
		}
		if f.Total != 1 {
			t.Errorf("total = %d, want 1", f.Total)
		}
		// Search is part of the always-applied base, so categories with no
		// match are gone rather than shown at 0 — they are not a filter the
		// visitor can undo by clicking.
		if _, ok := countOf(f.Categories, "beeswax"); ok {
			t.Error("beeswax still listed for a search that does not match it")
		}
		if got, ok := countOf(f.Categories, "royal-jelly"); !ok || got != 1 {
			t.Errorf("royal-jelly = %d (present=%v), want 1", got, ok)
		}
	})

	t.Run("nothing matches", func(t *testing.T) {
		f, err := s.CatalogFacets(ctx, domain.ProductFilter{Search: "bicycle"})
		if err != nil {
			t.Fatalf("CatalogFacets: %v", err)
		}
		if f.Total != 0 || len(f.Categories) != 0 || len(f.Benefits) != 0 {
			t.Errorf("empty result = %+v, want zeroes", f)
		}
		// min() over no rows is SQL NULL, which must not scan as an error.
		if f.PriceMinMinor != 0 || f.PriceMaxMinor != 0 {
			t.Errorf("price bounds = %d..%d, want 0..0", f.PriceMinMinor, f.PriceMaxMinor)
		}
	})
}

// The counts describe the whole filtered catalog, not the page on screen —
// which is the entire reason this is a separate query from the listing.
func TestCatalogFacets_IgnorePaging(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	f, err := s.CatalogFacets(ctx, domain.ProductFilter{Page: 2, PerPage: 1})
	if err != nil {
		t.Fatalf("CatalogFacets: %v", err)
	}
	if f.Total != 4 {
		t.Errorf("total = %d, want 4 regardless of the page", f.Total)
	}
}

// Inactive products are invisible to the storefront, so they must not be
// counted either — a sidebar reading "Honey 1" over an empty grid is worse
// than no sidebar.
func TestCatalogFacets_SkipInactiveProducts(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	if _, err := testPool.Exec(ctx,
		`UPDATE products SET is_active = FALSE WHERE slug = 'wax'`); err != nil {
		t.Fatal(err)
	}

	f, err := s.CatalogFacets(ctx, domain.ProductFilter{})
	if err != nil {
		t.Fatalf("CatalogFacets: %v", err)
	}
	if f.Total != 3 {
		t.Errorf("total = %d, want 3", f.Total)
	}
	for _, c := range f.Categories {
		if c.Slug == "beeswax" {
			t.Error("beeswax still listed with only an inactive product in it")
		}
	}
	// Wax was the cheapest thing in the shop; hiding it moves the slider.
	if f.PriceMinMinor != 1400 {
		t.Errorf("price floor = %d, want 1400 now that the 900 is hidden", f.PriceMinMinor)
	}
}
