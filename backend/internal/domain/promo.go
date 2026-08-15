package domain

import (
	"errors"
	"strings"
	"time"
)

// ── Promo codes ───────────────────────────────────────────────────────────

// The three kinds the design's promo box can resolve to. A percent code
// scales with the basket, a fixed code is a per-market amount (money, so it
// follows E5's per-market rule), and a free-shipping code waives the BASE
// rate — never the cold-chain surcharge, which no rule in this shop waives.
const (
	PromoPercent      = "percent"
	PromoFixed        = "fixed"
	PromoFreeShipping = "free_shipping"
)

// PromoValue is one market's money for a code — the promo_code_values row.
// Both fields are pointers because both are optional facts: a percent code
// has no amount anywhere, and a code without a floor has no minimum.
type PromoValue struct {
	AmountMinor      *int64
	MinSubtotalMinor *int64
}

// Promo is a code with everything needed to judge it: the rules from
// promo_codes, the per-market money, and two live counts the store fills in
// (total redemptions, and whether THIS shopper has used it). Carrying the
// counts as data keeps the judgement itself — Issue below — a pure function.
type Promo struct {
	ID      int64
	Code    string
	Kind    string
	Percent int // meaningful only for PromoPercent

	StartsAt *time.Time // nil = no lower bound
	EndsAt   *time.Time // nil = no upper bound

	MaxRedemptions *int // nil = uncapped
	Active         bool

	// Filled by the store for the asking user; see PromoForUser.
	Redemptions   int
	UsedByShopper bool

	Values map[Currency]PromoValue
}

// NormalizePromoCode is the one canonical form a code has: trimmed,
// uppercased. Applied at every boundary (apply endpoint, seed, lookup), so
// "  honey10 " and "HONEY10" are the same code everywhere — matching the
// upper() expression index that enforces uniqueness in the database.
func NormalizePromoCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

// Issue returns the validation code explaining why this promo cannot apply
// to a basket right now — or "" when it can. The same vocabulary as every
// other `fields` value: the frontend catalogue turns it into a sentence in
// the shopper's language.
//
// The order of checks is the order of honesty toward the customer:
// existence first, then time, then their own usage, then the global cap,
// then this basket. An inactive or not-yet-started code answers "unknown"
// on purpose — confirming that a disabled or unannounced code EXISTS is
// exactly the information a code-guesser is fishing for (the same
// reasoning as 404-not-403 for someone else's order).
//
// `now` is a parameter, not time.Now(): a function that reads the clock
// itself cannot be table-tested at a fixed instant, and expiry is exactly
// the kind of rule whose boundary deserves a test.
func (p Promo) Issue(now time.Time, currency Currency, subtotalMinor int64) string {
	if !p.Active || (p.StartsAt != nil && now.Before(*p.StartsAt)) {
		return ValidationPromoUnknown
	}
	if p.EndsAt != nil && now.After(*p.EndsAt) {
		return ValidationPromoExpired
	}
	if p.UsedByShopper {
		return ValidationPromoUsed
	}
	if p.MaxRedemptions != nil && p.Redemptions >= *p.MaxRedemptions {
		return ValidationPromoExhausted
	}

	v, hasValues := p.Values[currency]
	// A fixed code IS money, and money this shop has not priced in the
	// shopper's market cannot be granted there — refuse, never convert.
	if p.Kind == PromoFixed && (!hasValues || v.AmountMinor == nil) {
		return ValidationPromoNotInMarket
	}
	if hasValues && v.MinSubtotalMinor != nil && subtotalMinor < *v.MinSubtotalMinor {
		return ValidationPromoMinSubtotal
	}
	return ""
}

// ErrPromoInvalid is the checkout-time refusal: the cart carried a code
// that stopped being valid between apply and "Place the order" (expired,
// sold out, basket shrank below the floor). The API maps it to 409 — the
// customer saw a total that is no longer true, and silently charging a
// DIFFERENT total would be worse than asking them to look again.
var ErrPromoInvalid = errors.New("promo code is no longer valid")
