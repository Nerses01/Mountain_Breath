package domain

import "time"

// ── The hive club ─────────────────────────────────────────────────────────
//
// There is no membership table, and that is decision #36: the design's own
// sign-in screen defines the club as having an account — "Create an account
// — first order ships free", "8% less on every order after the first". Both
// perks are functions of ONE fact the orders table already holds (how many
// non-cancelled orders this customer has), so storing a tier would be a
// second copy of a derivable truth. Decision #66.
//
// The two perks are mutually exclusive by construction: order number one
// ships free, orders two onward are 8% off. Business facts, like VATPercent
// — a change to either deserves a commit, not a config edit.
const MemberDiscountPercent = 8

// ── The one calculator ────────────────────────────────────────────────────

// PriceInput is everything pricing a basket depends on, as plain values.
// The caller gathers them (the preview handler from live reads, CreateOrder
// from rows it has locked) and Price only computes — no database, no HTTP,
// no clock of its own. This is the layering rule at its sharpest: the
// arithmetic that decides what a customer pays is a pure function a table
// test can pin case by case.
type PriceInput struct {
	Currency      Currency
	SubtotalMinor int64
	HasColdChain  bool
	Rate          ShippingRate

	// Non-cancelled orders this customer already has. 0 = this is the first
	// order (free base delivery); ≥1 = member pricing (8% off).
	PriorOrders int

	// The code the cart carries, or nil. Price judges it itself (via Issue)
	// rather than trusting the caller to have done so: an invalid promo is
	// priced as NO promo, with the reason reported in Breakdown.PromoIssue —
	// the preview renders the reason, checkout refuses on it.
	Promo *Promo
	Now   time.Time
}

// Breakdown is the answer: every figure the summary card draws, plus the
// facts the cart page needs to explain itself (why shipping is free, how
// far the free-shipping bar has to go). It embeds OrderTotals so the store
// can stamp an order with exactly the numbers the customer was shown —
// same struct, no translation step to get wrong.
type Breakdown struct {
	OrderTotals
	Currency Currency

	// Why the base rate is (or is not) in ShippingMinor. FirstDeliveryFree
	// is the member perk; BaseShippingWaived is true for ANY reason (perk,
	// threshold, free-shipping promo) so the UI can label the line "Free".
	FirstDeliveryFree  bool
	BaseShippingWaived bool

	// How much more the basket needs before the threshold waives the base —
	// the "$8 away from free shipping" figure. nil when there is no
	// threshold in this market, or the base is already waived: a bar with
	// nothing to count toward is not shown, so it is not sent.
	RemainingForFreeShippingMinor *int64

	// The code that participated, and why it did not. PromoCode is set
	// whenever a code was attached (even an invalid one, so the UI can name
	// what it is complaining about); PromoIssue is "" when the code applied.
	PromoCode  string
	PromoIssue string
}

// percentOf takes pct% of amount in integer minor units, rounding half up —
// the same convention as ContainedVAT, and owned by one function for the
// same reason: however the rounding goes, it must go the same way on every
// screen, or the cart and the receipt disagree by a minor unit.
func percentOf(amountMinor int64, pct int) int64 {
	return (amountMinor*int64(pct) + 50) / 100
}

// Price is the single source of truth for what a basket costs. Cart page,
// checkout preview and order creation all call it — one function, three
// moments — which is what makes "cart, checkout and the order agree to the
// dram" a property of the design rather than a hope of the testing.
//
// Rule order, and why it is this order:
//
//  1. Shipping first. The base is waived by ANY of: the free-over threshold
//     (subtotal counts toward it), the first-delivery perk, a free-shipping
//     promo. The cold-chain surcharge survives all three — the chilled box
//     costs the family real money no matter whose promise waived the base.
//  2. Member discount: 8% of the subtotal, for customers past their first
//     order. Computed on the SHELF subtotal, which is what "8% less" means
//     on the sign-in screen.
//  3. Promo discount: percent codes also read the shelf subtotal (the two
//     discounts stack side by side, neither compounds on the other — a
//     compounding order would be an arbitrary hidden rule no screen could
//     explain); fixed codes grant their per-market amount, clamped so the
//     combined discount can never exceed the goods being discounted.
//  4. VAT is carved from what the customer actually pays for the goods
//     (subtotal − discount): the discounted price IS the price, and its
//     receipt line must contain the tax of the money that changed hands.
func Price(in PriceInput) Breakdown {
	b := Breakdown{Currency: in.Currency}
	b.SubtotalMinor = in.SubtotalMinor

	// ── 1. Shipping ──────────────────────────────────────────────────────
	base := in.Rate.BaseMinor
	if in.Rate.FreeOverMinor != nil && in.SubtotalMinor >= *in.Rate.FreeOverMinor {
		base = 0
	}
	if in.PriorOrders == 0 {
		b.FirstDeliveryFree = true
		base = 0
	}

	// ── 2 & 3. The two discounts ─────────────────────────────────────────
	if in.PriorOrders >= 1 {
		b.MemberDiscountMinor = percentOf(in.SubtotalMinor, MemberDiscountPercent)
	}

	if in.Promo != nil {
		b.PromoCode = in.Promo.Code
		b.PromoIssue = in.Promo.Issue(in.Now, in.Currency, in.SubtotalMinor)
		if b.PromoIssue == "" {
			switch in.Promo.Kind {
			case PromoPercent:
				b.PromoDiscountMinor = percentOf(in.SubtotalMinor, in.Promo.Percent)
			case PromoFixed:
				// Issue() == "" guarantees the amount exists in this market.
				b.PromoDiscountMinor = *in.Promo.Values[in.Currency].AmountMinor
			case PromoFreeShipping:
				base = 0
			}
			// A discount larger than the goods would push the total below
			// the shipping fee — the shop pays the customer to take honey.
			// Clamp the PROMO side: the member figure stays the stable 8%
			// the club promised, and the code absorbs the truncation.
			if room := in.SubtotalMinor - b.MemberDiscountMinor; b.PromoDiscountMinor > room {
				b.PromoDiscountMinor = room
			}
		}
	}

	b.BaseShippingWaived = in.Rate.BaseMinor > 0 && base == 0
	if in.HasColdChain {
		base += in.Rate.ColdChainSurchargeMinor
	}
	b.ShippingMinor = base

	// The progress bar's number: only meaningful while the base is still
	// being charged and a threshold exists to count toward.
	if in.Rate.FreeOverMinor != nil && !b.BaseShippingWaived {
		remaining := *in.Rate.FreeOverMinor - in.SubtotalMinor
		if remaining > 0 {
			b.RemainingForFreeShippingMinor = &remaining
		}
	}

	// ── 4. Assemble ──────────────────────────────────────────────────────
	b.DiscountMinor = b.MemberDiscountMinor + b.PromoDiscountMinor
	b.TaxMinor = ContainedVAT(in.SubtotalMinor - b.DiscountMinor)
	b.TotalMinor = in.SubtotalMinor + b.ShippingMinor - b.DiscountMinor
	return b
}
