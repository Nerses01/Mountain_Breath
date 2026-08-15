package domain

import "strings"

// ── Address ───────────────────────────────────────────────────────────────

// Address is where an order goes. The same shape serves two roles with very
// different lifetimes: a row in the customer's editable address book, and
// the frozen snapshot an order keeps of it. Go can share the struct because
// the difference is not in the fields — it is in who may change them, which
// the schema enforces (addresses is a table, the snapshot is columns on
// orders that nothing updates).
type Address struct {
	FirstName  string
	LastName   string
	Phone      string
	Street     string
	City       string
	PostalCode string
	Country    string
}

// ValidateAddress checks presence, not format — deliberately. Postal codes
// are four digits in Armenia, five in Russia, six in India; phone numbers
// are a standards swamp. A shop that rejects a real address because its
// pattern was written for another country loses the sale over its own
// cleverness. The one thing every field genuinely must be is non-blank.
//
// Field keys carry the `address.` prefix — the JSON path the checkout form
// posts, so the React form can attach each error to its input, exactly as
// `variants[0].sku` does for the admin.
func ValidateAddress(a Address) map[string]string {
	fields := make(map[string]string)
	check := func(key, value string) {
		if strings.TrimSpace(value) == "" {
			fields["address."+key] = ValidationRequired
		}
	}
	check("first_name", a.FirstName)
	check("last_name", a.LastName)
	check("phone", a.Phone)
	check("street", a.Street)
	check("city", a.City)
	check("postal_code", a.PostalCode)
	check("country", a.Country)
	return fields
}

// AddressEntry is one row of the account page's address book (E8): an
// Address plus its book-keeping. Embedding rather than repeating the seven
// fields — the entry IS an address, with a name tag and a default flag.
type AddressEntry struct {
	ID        int64
	Label     string // "Home", "Office" — the customer's word, may be empty
	IsDefault bool
	Address
}

// ── Payment ───────────────────────────────────────────────────────────────

// PaymentMethod is HOW the customer chose to pay. PaymentStatus (below) is
// WHETHER money has arrived — two separate facts, because a bank transfer
// is confirmed long before it is paid, and a cash order is delivered at the
// exact moment it stops being unpaid.
const (
	PayCard           = "card"
	PayBankTransfer   = "bank_transfer"
	PayCashOnDelivery = "cash_on_delivery"

	PaymentUnpaid   = "unpaid"
	PaymentPaid     = "paid"
	PaymentRefunded = "refunded"
)

// PaymentMethods in the order the design's three cards show them.
var PaymentMethods = []string{PayCard, PayBankTransfer, PayCashOnDelivery}

func ValidPaymentMethod(m string) bool {
	for _, pm := range PaymentMethods {
		if pm == m {
			return true
		}
	}
	return false
}

// ValidatePayment enforces the one cross-field rule the design states
// outright: "Cash — on delivery, AMD only". A courier collecting dollars in
// Yerevan is not a thing, so the rule lives HERE, in the domain, rather
// than as an if-statement in a handler — it is a business fact, and E5 made
// it expressible by giving every order a currency.
func ValidatePayment(method string, currency Currency) map[string]string {
	fields := make(map[string]string)
	if !ValidPaymentMethod(method) {
		fields["payment_method"] = ValidationInvalidPaymentMethod
		return fields
	}
	if method == PayCashOnDelivery && currency != CurrencyAMD {
		fields["payment_method"] = ValidationCashIsAMDOnly
	}
	return fields
}

// ── The money ─────────────────────────────────────────────────────────────

// OrderTotals is the five-figure breakdown the design's summary card draws,
// all in ONE currency (the order's) and all integer minor units.
//
// The invariant — Subtotal + Shipping − Discount = Total, with Tax contained
// in Subtotal rather than added — exists in three places on purpose: here
// (computed), in the CHECK constraint (enforced at rest), and in a
// table-driven test (documented). Three restatements of one sentence is the
// right amount of redundancy for the sentence that is the money.
type OrderTotals struct {
	SubtotalMinor int64
	ShippingMinor int64
	DiscountMinor int64
	TaxMinor      int64
	TotalMinor    int64

	// E7: the composition of DiscountMinor, because the receipt draws two
	// separate lines ("Hive club discount" and the promo by name) and one
	// number cannot say which promise produced it. Invariant, mirrored by
	// the orders_discount_split CHECK: Member + Promo = Discount.
	MemberDiscountMinor int64
	PromoDiscountMinor  int64
}

// VATPercent is Armenia's standard rate. It lives in the domain because the
// number is a business fact, not a config knob — when the law changes, the
// change deserves a commit with that law's name on it.
const VATPercent = 20

// ContainedVAT extracts the tax already inside a VAT-inclusive amount.
//
// This is the "tax contained in the price" model (§1.4): the shelf price IS
// the final price, and the tax figure is carved out of it for the invoice,
// never added on top. A 120 subtotal at 20% contains 20 of tax — not 24,
// which is what the naive subtotal×rate computes. Hence /(100+rate):
//
//	tax = subtotal × rate / (100 + rate)
//
// Integer arithmetic with round-half-up, no floats anywhere near it. The
// rounding direction is a convention, not a truth; what matters is that ONE
// function owns it, so every invoice line in the system rounds the same way.
func ContainedVAT(subtotalMinor int64) int64 {
	const rate = VATPercent
	return (subtotalMinor*rate + (100+rate)/2) / (100 + rate)
}

// ShippingRate is one market's delivery pricing — a row of shipping_rates.
type ShippingRate struct {
	BaseMinor               int64
	ColdChainSurchargeMinor int64
	// nil = this market has no free-shipping threshold. A pointer for the
	// same reason as the price filter bounds: 0 is not available to mean
	// "unset" when money is involved.
	FreeOverMinor *int64
}

// ShippingFor prices delivery of a given basket under this rate.
//
// Two rules, both from the design read literally:
//   - the BASE is waived once the subtotal clears the threshold — that is
//     what "$8 away from free shipping" is counting toward;
//   - the cold-chain surcharge is NEVER waived, because the mock's own cart
//     charges $6 "Chilled shipping" on a subtotal already past the
//     threshold. The chilled box costs the family real money either way.
func (r ShippingRate) ShippingFor(subtotalMinor int64, hasColdChain bool) int64 {
	shipping := r.BaseMinor
	if r.FreeOverMinor != nil && subtotalMinor >= *r.FreeOverMinor {
		shipping = 0
	}
	if hasColdChain {
		shipping += r.ColdChainSurchargeMinor
	}
	return shipping
}

// ComputeTotals was E6's assembler, superseded in E7 by Price (price.go) —
// the discount stopped being a pass-through parameter and became rules, and
// rules belong in the one calculator every screen shares. Deleted rather
// than kept as a second, discount-blind way to produce an OrderTotals.

// ── Quoting a cart ────────────────────────────────────────────────────────
//
// E6's QuoteCart lived here — per-market subtotal + shipping for a live
// basket. E7 deleted it: what delivery costs now depends on the CUSTOMER
// (the club waives a first order's base) and the cart's promo, so a quote
// computed from contents alone was one discount away from lying. Price
// (price.go) is the one calculator; the preview endpoint runs it per market.

// Upsell is the free-shipping banner's suggestion: one product that would
// close the remaining gap to the threshold — the design's "Add pollen ·
// $16". Name is already resolved into the shopper's language; PriceMinor is
// in the market being browsed.
type Upsell struct {
	// VariantID is what the banner's button actually adds — a cart line is
	// a variant, and a suggestion the client cannot act on in one click
	// would be a link wearing a button's clothes.
	VariantID  int64
	Slug       string
	Name       string
	PriceMinor int64
}

// ── The checkout request ──────────────────────────────────────────────────

// CheckoutInput is everything the CLIENT contributes to an order — and
// nothing else. Reading it is the security design of the phase: there is no
// money in it. The items come from the cart, the prices from
// variant_effective_prices, the shipping from shipping_rates, the currency
// from the edge — the client sends choices, the server computes every
// figure.
type CheckoutInput struct {
	Address            Address
	PaymentMethod      string
	DeliveryNote       string
	LeaveWithNeighbour bool
}

// ValidateCheckout gathers every field error at once, so the customer fixes
// one form, not one field per round trip.
func ValidateCheckout(in CheckoutInput, currency Currency) map[string]string {
	fields := ValidateAddress(in.Address)
	for k, v := range ValidatePayment(in.PaymentMethod, currency) {
		fields[k] = v
	}
	if len(in.DeliveryNote) > 500 {
		fields["delivery_note"] = ValidationTooLong
	}
	return fields
}
