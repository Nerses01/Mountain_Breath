package domain

import (
	"errors"
	"time"
)

const (
	OrderPending   = "pending"
	OrderConfirmed = "confirmed"
	OrderShipped   = "shipped"
	OrderDelivered = "delivered"
	OrderCancelled = "cancelled"
)

type Order struct {
	ID         int64
	UserID     int64
	UserEmail  string // filled only by admin queries
	Status     string
	TotalMinor int64
	CreatedAt  time.Time
	Items      []OrderItem

	// Currency is what the customer was actually charged in, and it is why
	// an order shows ONE price where a product card shows two. A cart is a
	// live thing that can be read in either market; an order is a fact about
	// a transaction that happened in exactly one of them. TotalMinor and
	// every item's PriceMinor are denominated in this.
	Currency Currency
	// FxRateUsed is how many units of Currency one unit of BaseCurrency
	// bought at checkout, or nil when the order was in the base currency and
	// no rate was involved. Snapshotted for the same reason prices are: it
	// answers "what was this order worth in dollars?" next year with the
	// rate that was true then, not with today's.
	//
	// A STRING, not a float64. The column is NUMERIC(18,8) — an exact
	// decimal — and float64 cannot hold 390.00000001 exactly. Carrying the
	// digits as text keeps the value the database stored, all the way to the
	// JSON, where it also dodges the fact that JSON.parse turns every number
	// into a double.
	FxRateUsed *string

	// E6. The frozen copy of where this order went — see the comment on
	// migration 000017 for why it is columns, not a reference into the
	// editable address book. Nil for orders that predate checkout-with-
	// address; the API requires it on every new one.
	ShipTo             *Address
	DeliveryNote       string
	LeaveWithNeighbour bool

	// The five-figure breakdown; Totals.TotalMinor and the legacy TotalMinor
	// field above carry the same number (the store scans both from one
	// column).
	Totals        OrderTotals
	PaymentMethod string
	PaymentStatus string

	// E7. The promo code this order redeemed — a snapshot of its TEXT, like
	// product names in order_items: the receipt keeps saying "HONEY10" even
	// if the family later renames or deletes the code. "" = no promo.
	PromoCode string
}

type OrderItem struct {
	ID         int64
	VariantID  int64
	Name       string // snapshot at purchase time
	Label      string
	PriceMinor int64
	Qty        int
}

var ErrInvalidTransition = errors.New("invalid order status transition")

// The order lifecycle as data: pending → confirmed → shipped → delivered,
// with cancellation possible while not yet shipped.
var orderTransitions = map[string][]string{
	OrderPending:   {OrderConfirmed, OrderCancelled},
	OrderConfirmed: {OrderShipped, OrderCancelled},
	OrderShipped:   {OrderDelivered},
}

func ValidOrderTransition(from, to string) bool {
	for _, allowed := range orderTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

func ValidOrderStatus(s string) bool {
	switch s {
	case OrderPending, OrderConfirmed, OrderShipped, OrderDelivered, OrderCancelled:
		return true
	}
	return false
}
