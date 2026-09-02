package domain_test

import (
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The sort whitelist is a SECURITY test, not a convenience one. ORDER BY
// cannot be a bound parameter, so the store picks a constant SQL fragment
// based on this value — which means anything ParseProductSort lets through
// reaches the query planner. The cases below are the ones an attacker would
// actually try.
func TestParseProductSort(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   domain.ProductSort
		wantOK bool
	}{
		{"popular", "popular", domain.SortPopular, true},
		{"rating", "rating", domain.SortRating, true},
		{"price ascending", "price_asc", domain.SortPriceAsc, true},
		{"price descending", "price_desc", domain.SortPriceDesc, true},
		{"newest", "newest", domain.SortNewest, true},

		// Everything else falls back to the default and reports false. Not
		// an error: a hand-edited URL should show the shop, not a 400.
		{"empty", "", domain.DefaultProductSort, false},
		{"unknown name", "cheapest", domain.DefaultProductSort, false},
		{"case matters", "POPULAR", domain.DefaultProductSort, false},
		{"sql fragment", "price_asc; DROP TABLE products", domain.DefaultProductSort, false},
		{"column injection", "p.sales_count DESC", domain.DefaultProductSort, false},
		{"prefix of a real value", "price", domain.DefaultProductSort, false},
		{"real value with a suffix", "newest--", domain.DefaultProductSort, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := domain.ParseProductSort(tc.input)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("ParseProductSort(%q) = %q, %v; want %q, %v",
					tc.input, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

// A zero-value filter must be usable — the store treats "no opinion" as
// "the default", the same contract EffectiveLocale has.
func TestProductFilterEffectiveSort(t *testing.T) {
	if got := (domain.ProductFilter{}).EffectiveSort(); got != domain.DefaultProductSort {
		t.Errorf("zero filter sort = %q; want %q", got, domain.DefaultProductSort)
	}
	f := domain.ProductFilter{Sort: domain.SortNewest}
	if got := f.EffectiveSort(); got != domain.SortNewest {
		t.Errorf("explicit sort = %q; want %q", got, domain.SortNewest)
	}
}

// ProductSorts is what the frontend's sort select is built from, so a value
// present in one list and missing from the other would ship a menu entry
// that silently does nothing.
func TestProductSortsCoversEveryConstant(t *testing.T) {
	for _, s := range []domain.ProductSort{
		domain.SortPopular, domain.SortRating,
		domain.SortPriceAsc, domain.SortPriceDesc, domain.SortNewest,
	} {
		parsed, ok := domain.ParseProductSort(string(s))
		if !ok || parsed != s {
			t.Errorf("constant %q is not in ProductSorts", s)
		}
	}
	// This count is a deliberate tripwire, and E4 tripped it: adding
	// `rating` failed here until the list above was updated too, which is
	// the point — a sort in the whitelist but missing from the frontend's
	// select is a menu entry that silently does nothing.
	if len(domain.ProductSorts) != 5 {
		t.Errorf("ProductSorts has %d entries; update this test with the new sort",
			len(domain.ProductSorts))
	}
}

// "Most loved" keeps meaning sales, and rating is its own sort. Pinned as a
// test because it is a PRODUCT decision that lives in a constant: an average
// over few reviews is violently unstable, so making it the default would let
// one five-star review outrank a jar that has sold 148 times, and the front
// page would reshuffle on every submission.
func TestDefaultSortIsPopularityNotRating(t *testing.T) {
	if domain.DefaultProductSort != domain.SortPopular {
		t.Errorf("default sort = %q, want popular", domain.DefaultProductSort)
	}
	if domain.DefaultProductSort == domain.SortRating {
		t.Error("rating must not be the default — see the note on DefaultProductSort")
	}
}
