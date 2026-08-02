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

type Product struct {
	ID          int64
	CategoryID  int64
	Slug        string
	Name        string
	Description string
	ImageURL    string
	IsActive    bool
	CreatedAt   time.Time
	Variants    []ProductVariant
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
}

func (f ProductFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
}

// ValidateProduct checks a product (with its variants) before creation.
// Field keys use the JSON path convention (variants[0].sku) so the frontend
// can attach errors to the right form input.
func ValidateProduct(p Product) map[string]string {
	fields := make(map[string]string)

	if strings.TrimSpace(p.Name) == "" {
		fields["name"] = "required"
	}
	switch {
	case p.Slug == "":
		fields["slug"] = "required"
	case !slugRe.MatchString(p.Slug):
		fields["slug"] = "must be lowercase letters/digits separated by dashes"
	}
	if p.CategoryID <= 0 {
		fields["category_id"] = "required"
	}
	if len(p.Variants) == 0 {
		fields["variants"] = "at least one variant is required"
	}

	for i, v := range p.Variants {
		if strings.TrimSpace(v.SKU) == "" {
			fields[fmt.Sprintf("variants[%d].sku", i)] = "required"
		}
		if strings.TrimSpace(v.Label) == "" {
			fields[fmt.Sprintf("variants[%d].label", i)] = "required"
		}
		if v.PriceMinor <= 0 {
			fields[fmt.Sprintf("variants[%d].price_minor", i)] = "must be positive"
		}
		if v.StockQty < 0 {
			fields[fmt.Sprintf("variants[%d].stock_qty", i)] = "must not be negative"
		}
	}
	return fields
}
