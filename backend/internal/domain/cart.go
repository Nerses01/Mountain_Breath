package domain

import (
	"errors"
	"fmt"
)

// CartItem is a cart line enriched with live product data for display —
// current price and stock, joined in by the store.
type CartItem struct {
	VariantID   int64
	ProductName string
	ProductSlug string
	Label       string
	StockQty    int
	Qty         int

	// IsColdChain rides along from the product (E3) because shipping needs
	// it: one chilled jar anywhere in the basket adds the cold-chain
	// surcharge to the whole parcel.
	IsColdChain bool

	// PriceMinor is the unit price in the currency this request resolved to
	// — the number the arithmetic on this screen uses.
	PriceMinor int64
	// Prices is the same unit price in every market, for the design's second,
	// muted line under the primary one. E5 made the cart dual-currency; see
	// Money for why both are carried rather than one being converted.
	Prices Money
}

func (c CartItem) LineTotalMinor() int64 {
	return c.PriceMinor * int64(c.Qty)
}

// LineTotals is LineTotalMinor for every market at once.
func (c CartItem) LineTotals() Money {
	return c.Prices.Scaled(c.Qty)
}

func CartTotalMinor(items []CartItem) int64 {
	var total int64
	for _, it := range items {
		total += it.LineTotalMinor()
	}
	return total
}

// CartTotals sums the basket independently in each currency.
//
// Note what this function does NOT do: convert. It never looks at an
// exchange rate, because every currency's column of numbers is already
// exact. See Money.AddTo for why that matters more than it looks.
func CartTotals(items []CartItem) Money {
	totals := make(Money, len(Currencies))
	if len(items) == 0 {
		for _, c := range Currencies {
			totals[c] = 0
		}
		return totals
	}

	// Seed from the first line, then INTERSECT. A currency that any single
	// line cannot be priced in is dropped from the total entirely, rather
	// than summed over the lines that can — a subtotal silently missing one
	// item is worse than no subtotal, because it is a wrong number that
	// looks right. (Deleting from a map while ranging over it is defined
	// behaviour in Go: an entry removed before it is reached is simply not
	// produced. C++ would need the erase-returns-next-iterator dance.)
	items[0].LineTotals().AddTo(totals)
	for _, it := range items[1:] {
		line := it.LineTotals()
		for c := range totals {
			minor, ok := line[c]
			if !ok {
				delete(totals, c)
				continue
			}
			totals[c] += minor
		}
	}
	return totals
}

var (
	ErrEmptyCart         = errors.New("cart is empty")
	ErrInsufficientStock = errors.New("insufficient stock")
)

// StockShortError is ErrInsufficientStock with the DATA attached: which line
// cannot be fulfilled and how many are actually left. A struct rather than a
// formatted string because the storefront speaks three languages — the API
// hands these fields to the client, which composes the sentence; prose baked
// in here could only ever be English.
//
// Unwrap returns the sentinel, so every existing errors.Is(err,
// ErrInsufficientStock) check — the API's status mapping, the tests — keeps
// working unchanged. This is Go's version of a derived exception type:
// callers who only care "was it a stock problem?" match the base, callers
// who want the numbers errors.As the concrete type.
type StockShortError struct {
	Name      string
	Label     string
	Available int
}

func (e *StockShortError) Error() string {
	return fmt.Sprintf("insufficient stock: %s (%s), %d available", e.Name, e.Label, e.Available)
}

func (e *StockShortError) Unwrap() error { return ErrInsufficientStock }
