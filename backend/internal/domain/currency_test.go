package domain_test

import (
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

func TestParseCurrency(t *testing.T) {
	tests := []struct {
		in    string
		want  domain.Currency
		valid bool
	}{
		{"USD", domain.CurrencyUSD, true},
		{"AMD", domain.CurrencyAMD, true},
		// A person typing a query string, not an attack.
		{"usd", domain.CurrencyUSD, true},
		{"aMd", domain.CurrencyAMD, true},
		// Unknown values fall back rather than erroring, and the caller is
		// told so by the second return value.
		{"EUR", domain.DefaultCurrency, false},
		{"", domain.DefaultCurrency, false},
		{"'; DROP TABLE variant_prices; --", domain.DefaultCurrency, false},
	}
	for _, tc := range tests {
		got, ok := domain.ParseCurrency(tc.in)
		if got != tc.want || ok != tc.valid {
			t.Errorf("ParseCurrency(%q) = (%v, %v), want (%v, %v)", tc.in, got, ok, tc.want, tc.valid)
		}
	}
}

func TestCurrencyForLocale(t *testing.T) {
	if got := domain.CurrencyForLocale(domain.LocaleHY); got != domain.CurrencyAMD {
		t.Errorf("hy → %v, want AMD", got)
	}
	// Russian is deliberately NOT mapped to drams: a Russian speaker may be
	// reading from anywhere. Language is a weak proxy for market, and the
	// shop only guesses where it is confident.
	for _, l := range []domain.Locale{domain.LocaleEN, domain.LocaleRU} {
		if got := domain.CurrencyForLocale(l); got != domain.DefaultCurrency {
			t.Errorf("%v → %v, want the default", l, got)
		}
	}
}

func TestCartTotalsSumEachMarketIndependently(t *testing.T) {
	// Three lines whose AMD prices are NOT 390× their USD prices — which is
	// the point of per-market pricing, and the reason the two columns have
	// to be added up separately.
	items := []domain.CartItem{
		{Qty: 1, Prices: domain.Money{domain.CurrencyUSD: 3200, domain.CurrencyAMD: 15300}},
		{Qty: 2, Prices: domain.Money{domain.CurrencyUSD: 1400, domain.CurrencyAMD: 6700}},
		{Qty: 1, Prices: domain.Money{domain.CurrencyUSD: 1900, domain.CurrencyAMD: 9100}},
	}

	totals := domain.CartTotals(items)
	if got := totals[domain.CurrencyUSD]; got != 7900 {
		t.Errorf("USD total = %d, want 7900", got)
	}
	if got := totals[domain.CurrencyAMD]; got != 37800 {
		t.Errorf("AMD total = %d, want 37800", got)
	}

	// THE LESSON. Converting the dollar total at the same rate that produced
	// none of these prices lands somewhere else entirely — 30,810 against
	// 37,800. Even with prices that WERE derived from a rate, rounding each
	// line and then summing is not the same as summing and then rounding.
	// This is why the store never converts a total, and why every currency
	// gets its own column of integers.
	const rate = 390
	if converted := totals[domain.CurrencyUSD] * rate / 100; converted == totals[domain.CurrencyAMD] {
		t.Fatalf("converting the USD total happened to equal the AMD total (%d) — "+
			"pick fixture prices that are not a fixed multiple, or the test proves nothing",
			converted)
	}
}

func TestCartTotalsDropAMarketOneLineCannotBePricedIn(t *testing.T) {
	items := []domain.CartItem{
		{Qty: 1, Prices: domain.Money{domain.CurrencyUSD: 1400, domain.CurrencyAMD: 6700}},
		// No AMD price on this one: the jar has a dollar shelf price and no
		// rate on file.
		{Qty: 3, Prices: domain.Money{domain.CurrencyUSD: 900}},
	}

	totals := domain.CartTotals(items)
	if got := totals[domain.CurrencyUSD]; got != 4100 {
		t.Errorf("USD total = %d, want 4100", got)
	}
	// 6,700 would be a subtotal that silently omits an item — a wrong number
	// that looks right. Absent is the honest answer.
	if _, ok := totals[domain.CurrencyAMD]; ok {
		t.Errorf("AMD total present (%d), want it dropped entirely", totals[domain.CurrencyAMD])
	}
}

func TestCartTotalsEmptyCartIsZeroEverywhere(t *testing.T) {
	totals := domain.CartTotals(nil)
	for _, c := range domain.Currencies {
		if got, ok := totals[c]; !ok || got != 0 {
			t.Errorf("%v = (%d, present=%v), want (0, true)", c, got, ok)
		}
	}
}

func TestMoneyScaled(t *testing.T) {
	unit := domain.Money{domain.CurrencyUSD: 1400, domain.CurrencyAMD: 6700}
	line := unit.Scaled(3)
	if line[domain.CurrencyUSD] != 4200 || line[domain.CurrencyAMD] != 20100 {
		t.Errorf("Scaled(3) = %v", line)
	}
	// Scaled must not mutate the receiver — the unit price is shared with
	// the response the caller is still building.
	if unit[domain.CurrencyUSD] != 1400 {
		t.Errorf("Scaled mutated its receiver: %v", unit)
	}
}
