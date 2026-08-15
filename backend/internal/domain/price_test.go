package domain_test

import (
	"testing"
	"time"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The fixed instant every case is judged at — the reason Promo.Issue and
// Price take `now` as a parameter instead of reading the clock is exactly
// so a test can hold time still.
var testNow = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

func i64(n int64) *int64 { return &n }
func iptr(n int) *int    { return &n }
func at(t time.Time) *time.Time {
	return &t
}

// The standard USD rate from migration 000017's bootstrap rows: $4 base,
// $6 cold-chain surcharge, base free over $70.
var usdRate = domain.ShippingRate{
	BaseMinor:               400,
	ColdChainSurchargeMinor: 600,
	FreeOverMinor:           i64(7000),
}

func percent10() *domain.Promo {
	return &domain.Promo{
		ID: 1, Code: "HONEY10", Kind: domain.PromoPercent, Percent: 10, Active: true,
	}
}

func fixed(amountMinor int64) *domain.Promo {
	return &domain.Promo{
		ID: 2, Code: "FIVER", Kind: domain.PromoFixed, Active: true,
		Values: map[domain.Currency]domain.PromoValue{
			domain.CurrencyUSD: {AmountMinor: i64(amountMinor)},
		},
	}
}

func freeShip() *domain.Promo {
	return &domain.Promo{
		ID: 3, Code: "FREESHIP", Kind: domain.PromoFreeShipping, Active: true,
	}
}

func TestPrice(t *testing.T) {
	tests := []struct {
		name string
		in   domain.PriceInput

		wantShipping, wantMember, wantPromo, wantTax, wantTotal int64
		wantFirstFree, wantWaived                               bool
		wantRemaining                                           *int64 // nil = no bar
		wantIssue                                               string
	}{
		{
			// "First order ships free" — the sign-in screen's promise. The
			// customer still sees the free-shipping bar? No: the base is
			// waived, so there is nothing to count toward and no Remaining.
			name: "first order: base waived, no member discount",
			in: domain.PriceInput{
				Currency: domain.CurrencyUSD, SubtotalMinor: 6200,
				Rate: usdRate, PriorOrders: 0, Now: testNow,
			},
			wantShipping: 0, wantMember: 0, wantPromo: 0,
			wantTax: 1033, wantTotal: 6200,
			wantFirstFree: true, wantWaived: true, wantRemaining: nil,
		},
		{
			// "8% less on every order after the first". 8% of 6200 = 496.
			// VAT is carved from what is actually paid for the goods:
			// ContainedVAT(6200 − 496 = 5704) = 951.
			name: "second order: member 8%, base charged, bar shows $8 gap",
			in: domain.PriceInput{
				Currency: domain.CurrencyUSD, SubtotalMinor: 6200,
				Rate: usdRate, PriorOrders: 1, Now: testNow,
			},
			wantShipping: 400, wantMember: 496, wantPromo: 0,
			wantTax: 951, wantTotal: 6104,
			wantRemaining: i64(800), // the design's "$8 away from free shipping"
		},
		{
			// The chilled parcel: base + surcharge, and the member discount
			// never touches shipping — 8% of the SUBTOTAL, as advertised.
			name: "member with cold chain pays base plus surcharge",
			in: domain.PriceInput{
				Currency: domain.CurrencyUSD, SubtotalMinor: 6200, HasColdChain: true,
				Rate: usdRate, PriorOrders: 3, Now: testNow,
			},
			wantShipping: 1000, wantMember: 496, wantPromo: 0,
			wantTax: 951, wantTotal: 6704,
			wantRemaining: i64(800),
		},
		{
			// Threshold crossed: the base goes, the surcharge SURVIVES —
			// §1.4's cart read literally, now with the member stacked on top.
			name: "over threshold, cold chain: only the surcharge",
			in: domain.PriceInput{
				Currency: domain.CurrencyUSD, SubtotalMinor: 9000, HasColdChain: true,
				Rate: usdRate, PriorOrders: 1, Now: testNow,
			},
			wantShipping: 600, wantMember: 720, wantPromo: 0,
			wantTax: 1380, wantTotal: 8880,
			wantWaived: true, wantRemaining: nil,
		},
		{
			// The two discounts stack SIDE BY SIDE: both read the shelf
			// subtotal, neither compounds on the other. 496 + 620, not
			// 496 + 570 (10% of the already-membered 5704).
			name: "member plus percent promo stack without compounding",
			in: domain.PriceInput{
				Currency: domain.CurrencyUSD, SubtotalMinor: 6200,
				Rate: usdRate, PriorOrders: 1, Promo: percent10(), Now: testNow,
			},
			wantShipping: 400, wantMember: 496, wantPromo: 620,
			wantTax: 847, wantTotal: 5484,
			wantRemaining: i64(800),
		},
		{
			// A fixed code bigger than the basket: the promo absorbs the
			// clamp (member keeps its promised 8%), the goods bottom out at
			// zero, and the customer still pays the shipping — the shop
			// never pays anyone to take honey.
			name: "oversized fixed promo clamps to what the goods allow",
			in: domain.PriceInput{
				Currency: domain.CurrencyUSD, SubtotalMinor: 1000,
				Rate: usdRate, PriorOrders: 1, Promo: fixed(50_00), Now: testNow,
			},
			wantShipping: 400, wantMember: 80, wantPromo: 920,
			wantTax: 0, wantTotal: 400,
			wantRemaining: i64(6000),
		},
		{
			// FREESHIP waives the base like the threshold does; the chilled
			// surcharge survives every kind of free shipping equally.
			name: "free-shipping promo: base gone, surcharge survives",
			in: domain.PriceInput{
				Currency: domain.CurrencyUSD, SubtotalMinor: 6200, HasColdChain: true,
				Rate: usdRate, PriorOrders: 2, Promo: freeShip(), Now: testNow,
			},
			wantShipping: 600, wantMember: 496, wantPromo: 0,
			wantTax: 951, wantTotal: 6304,
			wantWaived: true, wantRemaining: nil,
		},
		{
			// An expired code prices as NO code — with the reason reported,
			// not swallowed. The preview shows it; checkout refuses on it.
			name: "expired promo is priced out with its reason",
			in: domain.PriceInput{
				Currency: domain.CurrencyUSD, SubtotalMinor: 6200,
				Rate: usdRate, PriorOrders: 1, Now: testNow,
				Promo: &domain.Promo{
					ID: 9, Code: "AUGUST", Kind: domain.PromoPercent, Percent: 15,
					Active: true, EndsAt: at(testNow.Add(-time.Hour)),
				},
			},
			wantShipping: 400, wantMember: 496, wantPromo: 0,
			wantTax: 951, wantTotal: 6104,
			wantRemaining: i64(800),
			wantIssue:     domain.ValidationPromoExpired,
		},
		{
			// No threshold in this market (a rate row with NULL
			// free_over_minor): no bar, ever — nil, not a huge number.
			name: "market without a threshold has no progress bar",
			in: domain.PriceInput{
				Currency: domain.CurrencyAMD, SubtotalMinor: 1_000_000,
				Rate: domain.ShippingRate{BaseMinor: 1900}, PriorOrders: 1, Now: testNow,
			},
			wantShipping: 1900, wantMember: 80_000, wantPromo: 0,
			wantTax: 153_333, wantTotal: 921_900,
			wantRemaining: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := domain.Price(tc.in)

			if b.ShippingMinor != tc.wantShipping {
				t.Errorf("shipping = %d, want %d", b.ShippingMinor, tc.wantShipping)
			}
			if b.MemberDiscountMinor != tc.wantMember {
				t.Errorf("member discount = %d, want %d", b.MemberDiscountMinor, tc.wantMember)
			}
			if b.PromoDiscountMinor != tc.wantPromo {
				t.Errorf("promo discount = %d, want %d", b.PromoDiscountMinor, tc.wantPromo)
			}
			if b.TaxMinor != tc.wantTax {
				t.Errorf("tax = %d, want %d", b.TaxMinor, tc.wantTax)
			}
			if b.TotalMinor != tc.wantTotal {
				t.Errorf("total = %d, want %d", b.TotalMinor, tc.wantTotal)
			}
			if b.FirstDeliveryFree != tc.wantFirstFree {
				t.Errorf("first delivery free = %v", b.FirstDeliveryFree)
			}
			if b.BaseShippingWaived != tc.wantWaived {
				t.Errorf("base waived = %v", b.BaseShippingWaived)
			}
			if b.PromoIssue != tc.wantIssue {
				t.Errorf("promo issue = %q, want %q", b.PromoIssue, tc.wantIssue)
			}

			switch {
			case tc.wantRemaining == nil && b.RemainingForFreeShippingMinor != nil:
				t.Errorf("unexpected remaining %d", *b.RemainingForFreeShippingMinor)
			case tc.wantRemaining != nil && b.RemainingForFreeShippingMinor == nil:
				t.Errorf("remaining = nil, want %d", *tc.wantRemaining)
			case tc.wantRemaining != nil && *b.RemainingForFreeShippingMinor != *tc.wantRemaining:
				t.Errorf("remaining = %d, want %d", *b.RemainingForFreeShippingMinor, *tc.wantRemaining)
			}

			// The three invariants, restated on EVERY case so a future rule
			// change cannot break the books without breaking a test: the
			// balance the CHECK constraint enforces at rest, the discount
			// split, and tax containment.
			if b.SubtotalMinor+b.ShippingMinor-b.DiscountMinor != b.TotalMinor {
				t.Error("totals do not balance")
			}
			if b.MemberDiscountMinor+b.PromoDiscountMinor != b.DiscountMinor {
				t.Error("discount split does not sum")
			}
			if b.TaxMinor > b.SubtotalMinor {
				t.Error("contained tax exceeds what contains it")
			}
			if b.DiscountMinor > b.SubtotalMinor {
				t.Error("discount exceeds the goods")
			}
		})
	}
}

func TestPromoIssue(t *testing.T) {
	valid := domain.Promo{
		Code: "HONEY10", Kind: domain.PromoPercent, Percent: 10, Active: true,
		StartsAt: at(testNow.Add(-24 * time.Hour)), EndsAt: at(testNow.Add(24 * time.Hour)),
		MaxRedemptions: iptr(100), Redemptions: 40,
	}
	if issue := valid.Issue(testNow, domain.CurrencyUSD, 6200); issue != "" {
		t.Fatalf("valid promo refused: %q", issue)
	}

	tests := []struct {
		name   string
		mutate func(*domain.Promo)
		want   string
	}{
		// Inactive and not-yet-started both answer "unknown" — a disabled
		// or unannounced code must be indistinguishable from a nonexistent
		// one, or the error message becomes a code-existence oracle.
		{"inactive", func(p *domain.Promo) { p.Active = false }, domain.ValidationPromoUnknown},
		{"not started yet", func(p *domain.Promo) { p.StartsAt = at(testNow.Add(time.Hour)) }, domain.ValidationPromoUnknown},
		{"expired", func(p *domain.Promo) { p.EndsAt = at(testNow.Add(-time.Hour)) }, domain.ValidationPromoExpired},
		{"already used by this shopper", func(p *domain.Promo) { p.UsedByShopper = true }, domain.ValidationPromoUsed},
		{"globally exhausted", func(p *domain.Promo) { p.Redemptions = 100 }, domain.ValidationPromoExhausted},
		{
			// The boundary case the lock exists for: at max − 1 the code is
			// still grantable; the race test proves only one checkout gets it.
			"one redemption left is still valid",
			func(p *domain.Promo) { p.Redemptions = 99 },
			"",
		},
		{
			"fixed code with no amount in this market",
			func(p *domain.Promo) {
				p.Kind = domain.PromoFixed
				p.Percent = 0
				p.Values = map[domain.Currency]domain.PromoValue{
					domain.CurrencyAMD: {AmountMinor: i64(200000)},
				}
			},
			domain.ValidationPromoNotInMarket,
		},
		{
			"basket under the market's floor",
			func(p *domain.Promo) {
				p.Values = map[domain.Currency]domain.PromoValue{
					domain.CurrencyUSD: {MinSubtotalMinor: i64(10_000)},
				}
			},
			domain.ValidationPromoMinSubtotal,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := valid
			tc.mutate(&p)
			if got := p.Issue(testNow, domain.CurrencyUSD, 6200); got != tc.want {
				t.Errorf("issue = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNormalizePromoCode(t *testing.T) {
	if got := domain.NormalizePromoCode("  honey10 "); got != "HONEY10" {
		t.Errorf("normalize = %q", got)
	}
}
