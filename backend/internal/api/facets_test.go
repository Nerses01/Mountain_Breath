package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// Parsing a query string into a ProductFilter is API-layer work, so it is
// testable with no database at all: the fake store records what it was
// handed, and these assert on that.
func TestProductFilterParsing(t *testing.T) {
	tests := []struct {
		name  string
		query string
		check func(t *testing.T, f domain.ProductFilter)
	}{
		{
			name:  "repeated benefit params become a slice",
			query: "?benefit=energy&benefit=skin",
			check: func(t *testing.T, f domain.ProductFilter) {
				if len(f.BenefitSlugs) != 2 ||
					f.BenefitSlugs[0] != "energy" || f.BenefitSlugs[1] != "skin" {
					t.Errorf("benefits = %v, want [energy skin]", f.BenefitSlugs)
				}
			},
		},
		{
			name:  "no benefit param is an empty selection, not a filter",
			query: "",
			check: func(t *testing.T, f domain.ProductFilter) {
				if len(f.BenefitSlugs) != 0 {
					t.Errorf("benefits = %v, want none", f.BenefitSlugs)
				}
			},
		},
		{
			name:  "price bounds",
			query: "?min_price=900&max_price=3200",
			check: func(t *testing.T, f domain.ProductFilter) {
				if f.PriceMinMinor == nil || *f.PriceMinMinor != 900 {
					t.Errorf("min = %v, want 900", f.PriceMinMinor)
				}
				if f.PriceMaxMinor == nil || *f.PriceMaxMinor != 3200 {
					t.Errorf("max = %v, want 3200", f.PriceMaxMinor)
				}
			},
		},
		{
			// nil, not 0: a floor of zero would still be a filter, and
			// "unset" has to be distinguishable from "free".
			name:  "unparseable price is no bound at all",
			query: "?min_price=cheap&max_price=",
			check: func(t *testing.T, f domain.ProductFilter) {
				if f.PriceMinMinor != nil || f.PriceMaxMinor != nil {
					t.Errorf("bounds = %v/%v, want nil/nil", f.PriceMinMinor, f.PriceMaxMinor)
				}
			},
		},
		{
			name:  "negative price is dropped rather than passed down",
			query: "?min_price=-500",
			check: func(t *testing.T, f domain.ProductFilter) {
				if f.PriceMinMinor != nil {
					t.Errorf("min = %v, want nil", f.PriceMinMinor)
				}
			},
		},
		{
			// A slider bug or a hand-edited URL. Swapping returns the range
			// the visitor obviously meant; honouring it returns nothing.
			name:  "a reversed range is swapped",
			query: "?min_price=3000&max_price=1000",
			check: func(t *testing.T, f domain.ProductFilter) {
				if f.PriceMinMinor == nil || *f.PriceMinMinor != 1000 {
					t.Errorf("min = %v, want 1000", f.PriceMinMinor)
				}
				if f.PriceMaxMinor == nil || *f.PriceMaxMinor != 3000 {
					t.Errorf("max = %v, want 3000", f.PriceMaxMinor)
				}
			},
		},
		{
			name:  "sort",
			query: "?sort=price_desc",
			check: func(t *testing.T, f domain.ProductFilter) {
				if f.EffectiveSort() != domain.SortPriceDesc {
					t.Errorf("sort = %q, want price_desc", f.EffectiveSort())
				}
			},
		},
		{
			// The whitelist is what protects the ORDER BY; the handler must
			// fall back rather than pass the value through or 400.
			// Percent-encoded because a raw space is not a legal request
			// target — the payload reaches the handler decoded either way.
			name:  "an invalid sort falls back to the default",
			query: "?sort=%3B%20DROP%20TABLE%20products",
			check: func(t *testing.T, f domain.ProductFilter) {
				if f.EffectiveSort() != domain.DefaultProductSort {
					t.Errorf("sort = %q, want the default", f.EffectiveSort())
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeStore()
			srv := newTestServer(fake)

			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/products"+tc.query, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			tc.check(t, fake.lastFilter)
		})
	}
}

// Both endpoints read the same filter, so a divergence in parsing would show
// up as a sidebar whose counts do not describe the grid beside it.
func TestFacetsAndListingParseTheSameFilter(t *testing.T) {
	const query = "?category=honey&benefit=energy&benefit=skin&min_price=900&max_price=5800&q=jelly"

	fake := newFakeStore()
	srv := newTestServer(fake)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/products"+query, nil))
	fromListing := fake.lastFilter

	rec = httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/catalog/facets"+query, nil))
	fromFacets := fake.lastFilter

	if fromListing.CategorySlug != fromFacets.CategorySlug ||
		fromListing.Search != fromFacets.Search ||
		len(fromListing.BenefitSlugs) != len(fromFacets.BenefitSlugs) ||
		*fromListing.PriceMinMinor != *fromFacets.PriceMinMinor ||
		*fromListing.PriceMaxMinor != *fromFacets.PriceMaxMinor {
		t.Errorf("filters differ:\n listing = %+v\n facets  = %+v", fromListing, fromFacets)
	}

	// Paging is the one thing the facets endpoint must NOT carry — counts
	// describe the whole catalog, not the page.
	if fromFacets.Page != 0 || fromFacets.PerPage != 0 {
		t.Errorf("facets filter carries paging: page=%d per_page=%d",
			fromFacets.Page, fromFacets.PerPage)
	}
}

func TestFacetsResponseShape(t *testing.T) {
	fake := newFakeStore()
	fake.facets = domain.CatalogFacets{
		Categories:    []domain.FacetCount{{Slug: "honey", Name: "Honey", Count: 1}},
		Benefits:      []domain.FacetCount{{Slug: "energy", Name: "Energy", Count: 3}},
		Total:         4,
		PriceMinMinor: 900,
		PriceMaxMinor: 5800,
	}
	srv := newTestServer(fake)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/catalog/facets", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got struct {
		Categories []struct {
			Slug  string `json:"slug"`
			Name  string `json:"name"`
			Count int    `json:"count"`
		} `json:"categories"`
		Benefits []struct {
			Slug  string `json:"slug"`
			Count int    `json:"count"`
		} `json:"benefits"`
		Total         int   `json:"total"`
		PriceMinMinor int64 `json:"price_min_minor"`
		PriceMaxMinor int64 `json:"price_max_minor"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decoding: %v", err)
	}

	if len(got.Categories) != 1 || got.Categories[0].Slug != "honey" || got.Categories[0].Count != 1 {
		t.Errorf("categories = %+v", got.Categories)
	}
	if len(got.Benefits) != 1 || got.Benefits[0].Count != 3 {
		t.Errorf("benefits = %+v", got.Benefits)
	}
	if got.Total != 4 || got.PriceMinMinor != 900 || got.PriceMaxMinor != 5800 {
		t.Errorf("summary = %d, %d..%d", got.Total, got.PriceMinMinor, got.PriceMaxMinor)
	}
}

// The listing publishes the badge KEY, not English words — the client owns
// the wording. And it must not publish sales_count.
func TestProductResponseCarriesBadgeAndBenefitsButNotSalesCount(t *testing.T) {
	fake := newFakeStore()
	fake.products = []domain.Product{{
		ID: 1, Slug: "honey", Name: "Wildflower Honey",
		Badge: "best_seller", BadgeTone: "honey", SalesCount: 148,
		Benefits: []domain.Benefit{{Slug: "energy", Name: "Energy"}},
	}}
	srv := newTestServer(fake)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/products", nil))

	body := rec.Body.String()
	if !strings.Contains(body, `"badge":"best_seller"`) ||
		!strings.Contains(body, `"badge_tone":"honey"`) {
		t.Errorf("badge missing from response: %s", body)
	}
	if !strings.Contains(body, `"slug":"energy"`) {
		t.Errorf("benefits missing from response: %s", body)
	}
	if strings.Contains(body, "sales_count") || strings.Contains(body, "148") {
		t.Errorf("sales_count leaked into the public response: %s", body)
	}
}
