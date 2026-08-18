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

	// F2. The language the checkout happened in — Currency's sibling
	// snapshot, and for the same reason: a status-change email is sent
	// days later by the ADMIN's request, whose negotiated language is the
	// admin's. The customer's language is a fact about the order.
	Locale Locale

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

	// A2 (decision log #85). The recorded timeline, oldest first: one event
	// per status transition, from order_status_events. Status stays the
	// CURRENT state; this is how it got there and when. Orders created
	// before the events table carry only their backfilled `pending` event —
	// their later steps happened but were never recorded, and the tracker
	// shows them without dates rather than with invented ones.
	Events []OrderEvent

	// A2. True when any line is a cold-chain product — the canvas's
	// "chilled parcel" tag. Derived from the CURRENT product flags at read
	// time (like the cart's), not snapshotted: it labels the parcel's
	// handling, it is not part of the financial record.
	HasColdChain bool
}

// OrderEvent is one step of an order's recorded history.
type OrderEvent struct {
	Status    string
	CreatedAt time.Time
}

// A2 (decision log #86): why a reorder line could not be re-added in full.
// Codes, not sentences — the client translates them, the same contract as
// the checkout preview's promo_issue.
const (
	ReorderUnavailable = "unavailable"  // the product was retired since
	ReorderOutOfStock  = "out_of_stock" // nothing left to add
	ReorderReduced     = "reduced"      // added, but fewer than last time
)

// ReorderLine is one order line's fate when re-added to the cart.
type ReorderLine struct {
	Name  string // the order's SNAPSHOT name — what the customer remembers buying
	Label string
	Qty   int    // actually added to the cart; 0 when skipped
	Issue string // "" = added in full, else one of the Reorder* codes
}

// ReorderResult reports the whole merge: partial success is the expected
// case, not an error — an order from last month meeting today's stock.
type ReorderResult struct {
	Lines []ReorderLine
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
