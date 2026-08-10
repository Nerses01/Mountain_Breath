package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrNotFound is the generic "no such row" sentinel; the API layer maps it
// to 404. Defined here so no layer needs to import database packages to
// recognize it.
var ErrNotFound = errors.New("not found")

var (
	ErrSKUTaken          = errors.New("sku already exists")
	ErrVariantLabelTaken = errors.New("variant label already used for this product")
	ErrCategoryNotFound  = errors.New("no such category")
)

// ProductText is the translatable half of a product. The slug, SKU, price
// and stock are not in here on purpose — they mean the same thing in every
// language, and duplicating them per locale would create three ways to
// disagree about one fact.
type ProductText struct {
	Name        string
	Description string
}

type Product struct {
	ID         int64
	CategoryID int64
	Slug       string
	ImageURL   string
	IsActive   bool
	CreatedAt  time.Time
	Variants   []ProductVariant

	// The product's category, resolved on a read. The card's eyebrow shows
	// the NAME and the sidebar filters on the SLUG, and the list query
	// already joins categories — carrying both here saves every client a
	// second request and an id→name lookup it would have to keep in sync.
	CategorySlug string
	CategoryName string

	// Badge is a KEY ("best_seller"), not a sentence — the client owns the
	// wording in its three message catalogues. Empty means no badge.
	// BadgeTone is presentation only ("honey"/"dark"/"outline").
	Badge     string
	BadgeTone string

	// SalesCount is the denormalized popularity counter behind the "Most
	// loved" sort (migration 000010). Read-only above the store: it is
	// maintained by the checkout transaction, never set by a caller.
	SalesCount int

	// Benefits is the "Good for" taxonomy — many per product, hence a slice
	// where CategoryID is a single field.
	Benefits []Benefit

	// Name/Description are English on a write and the RESOLVED text for the
	// requested locale on a read — see the note on Category.Name.
	Name        string
	Description string

	// Translations holds non-default locales only; English is the pair above.
	Translations map[Locale]ProductText
}

type ProductVariant struct {
	ID         int64
	ProductID  int64
	SKU        string
	Label      string
	PriceMinor int64 // money in minor units (e.g. 180000 = 1800.00)
	StockQty   int
}

// ProductSort is the whitelist of orderings the Shop page's select offers.
//
// A string type rather than a bare string so the compiler stops a handler
// from passing an arbitrary value: the ONLY way to obtain one is
// ParseProductSort, which checks membership. That matters more than usual
// here — the value ends up choosing an ORDER BY clause, and ORDER BY cannot
// be a bound parameter (Postgres plans the sort at parse time, so `ORDER BY
// $1` sorts by the constant string $1, not by that column). Every other
// piece of user input in this file reaches SQL as a parameter; this one
// reaches it by selecting a compile-time constant, which is why the
// whitelist lives in the domain layer instead of being "validated" in the
// store next to the query.
type ProductSort string

const (
	SortPopular   ProductSort = "popular"    // sales_count, the "Most loved" default
	SortPriceAsc  ProductSort = "price_asc"  // cheapest variant, ascending
	SortPriceDesc ProductSort = "price_desc" // cheapest variant, descending
	SortNewest    ProductSort = "newest"     // created_at

	// DefaultProductSort is what an absent or unrecognised sort resolves to
	// — the design labels the select "Sort: Most loved".
	DefaultProductSort = SortPopular
)

// ProductSorts is the whole set, in the order the select lists them.
var ProductSorts = []ProductSort{SortPopular, SortPriceAsc, SortPriceDesc, SortNewest}

// ParseProductSort reports whether s named a real sort. Callers decide
// whether an unknown value is an error or simply falls back — the same
// contract as ParseLocale, deliberately, so the two read alike.
func ParseProductSort(s string) (ProductSort, bool) {
	for _, ps := range ProductSorts {
		if string(ps) == s {
			return ps, true
		}
	}
	return DefaultProductSort, false
}

// ProductFilter describes what a product listing should return.
type ProductFilter struct {
	CategorySlug    string // empty = all categories
	Search          string // empty = no text search
	IncludeInactive bool   // admin listings see deactivated products too
	Page            int    // 1-based
	PerPage         int
	// Locale selects which translation to display and which text search
	// configuration to stem the query with. Zero value means unset, not
	// invalid — see EffectiveLocale.
	Locale Locale

	// BenefitSlugs is an OR within the facet: picking Energy and Immunity
	// widens the result to products good for either. Filters from DIFFERENT
	// facets combine with AND (honey AND energy), which is the convention
	// every faceted shop uses — narrowing inside one group would make the
	// second click almost always return nothing.
	BenefitSlugs []string

	// Price bounds in minor units, nil = unbounded on that side. Pointers
	// rather than an int64 sentinel because 0 is a legitimate bound and
	// there is no spare value to mean "unset" — the same job std::optional
	// does in C++, done here by the fact that a pointer can be nil. They
	// also travel to pgx as SQL NULL without any conversion.
	PriceMinMinor *int64
	PriceMaxMinor *int64

	// Sort is empty when the caller does not care; see EffectiveSort.
	Sort ProductSort
}

func (f ProductFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
}

// EffectiveSort mirrors EffectiveLocale: a zero-value filter (a test, or a
// caller written before sorting existed) means "no opinion", not "invalid",
// and gets the default rather than an empty ORDER BY.
func (f ProductFilter) EffectiveSort() ProductSort {
	if f.Sort == "" {
		return DefaultProductSort
	}
	return f.Sort
}

// EffectiveLocale is the language to actually query in. A zero-value filter
// (a caller that predates translations, or a test that does not care) asks
// for no locale at all, and gets English rather than an empty string that
// would match no translation row and silently blank every name.
func (f ProductFilter) EffectiveLocale() Locale {
	if f.Locale == "" {
		return DefaultLocale
	}
	return f.Locale
}

// ValidateProduct checks a product (with its variants) before creation.
// Field keys use the JSON path convention (variants[0].sku) so the frontend
// can attach errors to the right form input.
func ValidateProduct(p Product) map[string]string {
	fields := make(map[string]string)

	if strings.TrimSpace(p.Name) == "" {
		fields["name"] = ValidationRequired
	}
	switch {
	case p.Slug == "":
		fields["slug"] = ValidationRequired
	case !slugRe.MatchString(p.Slug):
		fields["slug"] = ValidationSlugFormat
	}
	if p.CategoryID <= 0 {
		fields["category_id"] = ValidationRequired
	}
	if len(p.Variants) == 0 {
		fields["variants"] = ValidationVariantsRequired
	}

	for locale, text := range p.Translations {
		key := "translations." + string(locale)
		if strings.TrimSpace(text.Name) == "" {
			// Omitting a language is fine — it falls back to English. Sending
			// one with a blank name is not: that is a form bug, not a choice.
			fields[key+".name"] = ValidationRequired
		}
	}
	for k, v := range ValidateTranslationLocales("translations", p.Translations) {
		fields[k] = v
	}

	for i, v := range p.Variants {
		if strings.TrimSpace(v.SKU) == "" {
			fields[fmt.Sprintf("variants[%d].sku", i)] = ValidationRequired
		}
		if strings.TrimSpace(v.Label) == "" {
			fields[fmt.Sprintf("variants[%d].label", i)] = ValidationRequired
		}
		if v.PriceMinor <= 0 {
			fields[fmt.Sprintf("variants[%d].price_minor", i)] = ValidationPositive
		}
		if v.StockQty < 0 {
			fields[fmt.Sprintf("variants[%d].stock_qty", i)] = ValidationNotNegative
		}
	}
	return fields
}
