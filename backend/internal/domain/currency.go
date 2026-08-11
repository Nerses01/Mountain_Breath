package domain

import "errors"

// Currency is one of the markets the shop prices in. Deliberately the same
// shape as Locale: a string type whose only legal values come from a
// whitelist, because it too ends up selecting rows from the database and
// choosing what a visitor is charged.
//
// The Go set duplicates the `currencies` table, exactly as Locales
// duplicates the locale CHECK constraint. That duplication is a real cost —
// but it is a cost with a test attached: TestCurrenciesMatchTheDatabase in
// the store package fails if the two ever disagree. A comment saying "keep
// these in sync" is a wish; a failing test is a guarantee.
//
// Why Go needs the set at all, when the properties (symbol, exponent,
// rounding step) live only in SQL: the middleware has to reject
// ?currency=ZZZ on every request, and that answer must not cost a query.
// Everything else about a currency is used where the rounding happens —
// inside the database.
type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyAMD Currency = "AMD"

	// DefaultCurrency is what an unrecognised or absent request resolves to.
	DefaultCurrency = CurrencyUSD

	// BaseCurrency is the one prices are authored in and FX rates are quoted
	// against — `currencies.is_base` in SQL. It is special in a way the
	// others are not: a variant is REQUIRED to have a price in it, because
	// it is the only price the fallback conversion can start from.
	BaseCurrency = CurrencyUSD
)

// Currencies is the whole supported set, in the order the switcher lists
// them — mirroring currencies.sort_order.
var Currencies = []Currency{CurrencyUSD, CurrencyAMD}

// ErrPriceUnavailable means a variant cannot be priced in the requested
// currency at all: no shelf price, and no rate to convert one.
//
// This is a checkout error, not a browsing one. Reads degrade — a card with
// no AMD price simply shows no AMD line — because a shop that 500s over a
// missing secondary price is worse than one that shows a single price.
// Charging does not degrade: the alternative to failing here is billing
// someone zero.
var ErrPriceUnavailable = errors.New("no price available in this currency")

// ParseCurrency accepts any casing — ?currency=usd is a person typing, not
// an attack — and reports whether it named a currency the shop serves.
// Callers decide whether an unknown one is an error or falls back, the same
// contract as ParseLocale and ParseProductSort.
func ParseCurrency(s string) (Currency, bool) {
	if s == "" {
		return DefaultCurrency, false
	}
	up := upperASCII(s)
	for _, c := range Currencies {
		if string(c) == up {
			return c, true
		}
	}
	return DefaultCurrency, false
}

// CurrencyForLocale is the "where does this reader probably shop?" guess,
// used only as a fallback when nothing more explicit is known.
//
// Language is a poor proxy for market — a Russian speaker in Yerevan and one
// in Moscow read the same site — so this maps only the case the shop is
// confident about: the Armenian interface is for the Armenian market. Every
// other language gets the default and can switch.
func CurrencyForLocale(l Locale) Currency {
	if l == LocaleHY {
		return CurrencyAMD
	}
	return DefaultCurrency
}

func (c Currency) String() string { return string(c) }

// Money is an amount expressed in every market at once: currency → integer
// minor units. The map, rather than a single number plus a rate, is the
// point of the whole phase — there is no "the" price to convert from.
//
// Still integer minor units per entry (rule: money is never float), but the
// SCALE of those units now differs per currency: 1400 USD-minor is $14.00,
// 5460 AMD-minor is 5,460 ֏. Nothing may divide by 100 any more.
type Money map[Currency]int64

// Scaled returns the amount in every currency multiplied by n — a line
// total from a unit price.
func (m Money) Scaled(n int) Money {
	out := make(Money, len(m))
	for c, minor := range m {
		out[c] = minor * int64(n)
	}
	return out
}

// AddTo accumulates m into dst, per currency.
//
// THIS is the money lesson of E5, and it is worth stating plainly: totals
// are summed SEPARATELY IN EACH CURRENCY, never summed once and converted.
//
// Converting a total is not the same arithmetic as totalling conversions.
// Each per-market price is an independent, exact integer; adding them is
// exact. Converting introduces one rounding error, and rounding a sum is not
// the sum of roundings — three lines that each round up by 4 drams make a
// total that is 12 drams away from the converted subtotal, and a customer
// who adds up the line items on screen gets a different answer from the one
// the shop charges. Per-market prices make that impossible by construction.
func (m Money) AddTo(dst Money) {
	for c, minor := range m {
		dst[c] += minor
	}
}

// upperASCII is lowerASCII's twin — a currency code is only ever three
// ASCII letters, so the Unicode machinery in strings.ToUpper is not needed.
func upperASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - ('a' - 'A')
		}
	}
	return string(b)
}
