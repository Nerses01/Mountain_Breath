package domain

import (
	"errors"
	"time"
)

// ErrNotFound is the generic "no such row" sentinel; the API layer maps it
// to 404. Defined here so no layer needs to import database packages to
// recognize it.
var ErrNotFound = errors.New("not found")

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
	CategorySlug string // empty = all categories
	Page         int    // 1-based
	PerPage      int
}

func (f ProductFilter) Offset() int {
	return (f.Page - 1) * f.PerPage
}
