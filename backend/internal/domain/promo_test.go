package domain_test

import (
	"testing"
	"time"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// F2: the admin's promo form validation — a mirror of migration 000018's
// CHECK constraints in the fields vocabulary. Each case names the
// constraint it guards, so a schema change that loosens one is caught here.
func TestValidatePromoInput(t *testing.T) {
	pct := func(n int) *int { return &n }
	minor := func(n int64) *int64 { return &n }
	at := func(s string) *time.Time {
		ts, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return &ts
	}

	valid := domain.PromoInput{
		Code: "HONEY10", Kind: domain.PromoPercent, Percent: pct(10), Active: true,
	}

	t.Run("a valid percent code passes clean", func(t *testing.T) {
		if fields := domain.ValidatePromoInput(valid); len(fields) != 0 {
			t.Errorf("unexpected fields: %v", fields)
		}
	})

	t.Run("a valid fixed code needs an amount somewhere", func(t *testing.T) {
		in := domain.PromoInput{
			Code: "WELCOME", Kind: domain.PromoFixed,
			Values: map[domain.Currency]domain.PromoValue{
				domain.CurrencyUSD: {AmountMinor: minor(500)},
			},
		}
		if fields := domain.ValidatePromoInput(in); len(fields) != 0 {
			t.Errorf("unexpected fields: %v", fields)
		}
		in.Values = nil
		if fields := domain.ValidatePromoInput(in); fields["values"] != domain.ValidationRequired {
			t.Errorf("amount-less fixed code passed: %v", fields)
		}
	})

	cases := []struct {
		name  string
		mut   func(*domain.PromoInput)
		field string
	}{
		{"blank code (after trim)", func(in *domain.PromoInput) { in.Code = "   " }, "code"},
		{"unknown kind", func(in *domain.PromoInput) { in.Kind = "bogo" }, "kind"},
		{"percent kind without percent", func(in *domain.PromoInput) { in.Percent = nil }, "percent"},
		{"percent out of range", func(in *domain.PromoInput) { in.Percent = pct(101) }, "percent"},
		{"percent on a non-percent kind", func(in *domain.PromoInput) {
			in.Kind = domain.PromoFreeShipping
		}, "percent"},
		{"window inverted", func(in *domain.PromoInput) {
			in.StartsAt = at("2026-09-01T00:00:00Z")
			in.EndsAt = at("2026-08-01T00:00:00Z")
		}, "ends_at"},
		{"zero redemption cap", func(in *domain.PromoInput) { in.MaxRedemptions = pct(0) }, "max_redemptions"},
		{"unknown currency key", func(in *domain.PromoInput) {
			in.Values = map[domain.Currency]domain.PromoValue{"EUR": {}}
		}, "values.EUR"},
		{"amount on a percent code", func(in *domain.PromoInput) {
			in.Values = map[domain.Currency]domain.PromoValue{
				domain.CurrencyUSD: {AmountMinor: minor(500)},
			}
		}, "values.USD.amount_minor"},
		{"non-positive floor", func(in *domain.PromoInput) {
			in.Values = map[domain.Currency]domain.PromoValue{
				domain.CurrencyUSD: {MinSubtotalMinor: minor(0)},
			}
		}, "values.USD.min_subtotal_minor"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			in := valid
			tt.mut(&in)
			if fields := domain.ValidatePromoInput(in); fields[tt.field] == "" {
				t.Errorf("field %q not flagged: %v", tt.field, fields)
			}
		})
	}

	t.Run("half-open windows are legal", func(t *testing.T) {
		in := valid
		in.StartsAt = at("2026-09-01T00:00:00Z") // "until the jars run out"
		if fields := domain.ValidatePromoInput(in); len(fields) != 0 {
			t.Errorf("open-ended window flagged: %v", fields)
		}
	})
}
