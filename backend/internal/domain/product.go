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
}

func (f ProductFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
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
