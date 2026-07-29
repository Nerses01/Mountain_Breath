package domain

import "errors"

// CartItem is a cart line enriched with live product data for display —
// current price and stock, joined in by the store.
type CartItem struct {
	VariantID   int64
	ProductName string
	ProductSlug string
	Label       string
	PriceMinor  int64
	StockQty    int
	Qty         int
}

func (c CartItem) LineTotalMinor() int64 {
	return c.PriceMinor * int64(c.Qty)
}

func CartTotalMinor(items []CartItem) int64 {
	var total int64
	for _, it := range items {
		total += it.LineTotalMinor()
	}
	return total
}

var (
	ErrEmptyCart         = errors.New("cart is empty")
	ErrInsufficientStock = errors.New("insufficient stock")
)
