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
//
// The E3 notes are scalar fields of the product, so they live here beside
// name and description rather than in tables of their own — unlike the
// ordered collections below, which are genuinely child rows.
type ProductText struct {
	Name        string
	Description string

	Disclaimer   string // "Not a medicine. Avoid if you are allergic…"
	StorageNote  string // the Storage tab's paragraph
	HarvestNote  string // "June 2026, Hive 41"
	ShippingNote string // "Chilled, 2–4 days"
}

// ProductImage is one photo in the gallery. Unlike a highlight, most of this
// row is locale-invariant — the file, its position, whether it is the hero —
// so only Alt is resolved per language (migration 000011).
type ProductImage struct {
	ID        int64
	URL       string
	SortOrder int
	IsPrimary bool
	Alt       string
}

// ProductHighlight is one "What it does" bullet. The whole ROW is per-locale
// (migration 000012), so there is no id here worth exposing: a bullet's
// identity is its position within a language.
type ProductHighlight struct {
	SortOrder int
	Text      string
}

// ProductUsageCard is one of the Morning / Course / Pairs-with cards. Three
// fields rather than one blob because the design types each differently, and
// splitting prose in the frontend cannot be validated.
type ProductUsageCard struct {
	SortOrder int
	Kicker    string
	Title     string
	Body      string
}

// ProductEditorial is one language's worth of editorial content — what the
// admin form submits per locale tab, and what the store replaces as a unit.
//
// Replaced wholesale rather than edited row by row: the rows are keyed by
// position, so "the third bullet" has no identity to update once the admin
// reorders or deletes one. Sending the whole list and rewriting it inside a
// transaction is both simpler and the only version that cannot half-apply.
type ProductEditorial struct {
	Highlights []ProductHighlight
	UsageCards []ProductUsageCard
}

// ValidateEditorial rejects blank rows. A missing language is fine — the
// read falls back to English as a whole list — but an empty bullet inside a
// submitted list is a form bug, exactly like a blank translated name.
func ValidateEditorial(prefix string, e ProductEditorial) map[string]string {
	fields := make(map[string]string)
	for i, h := range e.Highlights {
		if strings.TrimSpace(h.Text) == "" {
			fields[fmt.Sprintf("%s.highlights[%d].text", prefix, i)] = ValidationRequired
		}
	}
	for i, c := range e.UsageCards {
		if strings.TrimSpace(c.Title) == "" {
			fields[fmt.Sprintf("%s.usage_cards[%d].title", prefix, i)] = ValidationRequired
		}
		if strings.TrimSpace(c.Body) == "" {
			fields[fmt.Sprintf("%s.usage_cards[%d].body", prefix, i)] = ValidationRequired
		}
	}
	return fields
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

	// Rating is the denormalized review aggregate (migration 000015), also
	// read-only above the store: it is recomputed from the reviews table
	// whenever a review's rating or status changes.
	Rating RatingSummary

	// CanReview answers "may the CURRENT viewer review this?", so the UI
	// does not have to reimplement the verified-purchase rule and guess.
	// Only populated by the detail read, and only for a signed-in viewer.
	CanReview bool

	// Benefits is the "Good for" taxonomy — many per product, hence a slice
	// where CategoryID is a single field.
	Benefits []Benefit

	// Locale-invariant product metadata (migration 000013).
	LabBatch    string
	IsColdChain bool

	// Editorial content, loaded only by the DETAIL read. The listing leaves
	// these empty on purpose: a card shows none of them, and fetching six
	// products' worth of bullets to render six cards is work nobody sees.
	Images     []ProductImage
	Highlights []ProductHighlight
	UsageCards []ProductUsageCard

	// Name/Description are English on a write and the RESOLVED text for the
	// requested locale on a read — see the note on Category.Name. The four
	// notes below follow exactly the same rule.
	Name         string
	Description  string
	Disclaimer   string
	StorageNote  string
	HarvestNote  string
	ShippingNote string

	// Translations holds non-default locales only; English is the pair above.
	Translations map[Locale]ProductText
}

// PrimaryImage is the gallery's hero, or the zero value when a product has
// no images at all. The database guarantees at most one is_primary per
// product (a partial unique index in 000011), so this cannot be ambiguous —
// but it CAN be absent, which is why the second return value exists rather
// than handing back a silently empty image.
func (p Product) PrimaryImage() (ProductImage, bool) {
	for _, img := range p.Images {
		if img.IsPrimary {
			return img, true
		}
	}
	// Falling back to the first image rather than nothing: an uncurated
	// gallery should still show a picture.
	if len(p.Images) > 0 {
		return p.Images[0], true
	}
	return ProductImage{}, false
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
	SortRating    ProductSort = "rating"     // rating_avg, then how many rated it
	SortPriceAsc  ProductSort = "price_asc"  // cheapest variant, ascending
	SortPriceDesc ProductSort = "price_desc" // cheapest variant, descending
	SortNewest    ProductSort = "newest"     // created_at

	// DefaultProductSort is what an absent or unrecognised sort resolves to
	// — the design labels the select "Sort: Most loved".
	//
	// E4 DECISION: "Most loved" keeps meaning sales_count, and rating gets
	// its OWN entry rather than taking this one over.
	//
	// Rating is the more literal reading of the words, and it is the wrong
	// default: an average over few reviews is violently unstable, so one
	// five-star review would outrank a jar that has sold 148 times, and the
	// shop's front page would reshuffle on every submission. Ranking by
	// rating honestly needs a Bayesian prior (weight each product's average
	// toward the catalog mean by how few ratings it has), which is real work
	// this six-product shop cannot justify — see the learning log.
	//
	// So: two sorts, each meaning exactly what it says. "Most loved" =
	// bought most; "Best rated" = rated highest, tie-broken by how many
	// people rated it, which is the cheap half of the same idea.
	DefaultProductSort = SortPopular
)

// ProductSorts is the whole set, in the order the select lists them.
var ProductSorts = []ProductSort{
	SortPopular, SortRating, SortPriceAsc, SortPriceDesc, SortNewest,
}

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
