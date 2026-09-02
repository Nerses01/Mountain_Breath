package domain_test

import (
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

func TestContainedVAT(t *testing.T) {
	// tax = subtotal × 20 / 120, round half up. The first case is the whole
	// model in one line: a 120 subtotal CONTAINS 20 of tax — the naive
	// subtotal×20% would say 24, which is the tax-added-on-top answer and
	// wrong under "Prices include VAT".
	tests := []struct {
		subtotal int64
		want     int64
	}{
		{120, 20},
		{0, 0},
		{6200, 1033},  // $62.00 → $10.33 (61.99̄ rounds up at .3)
		{28700, 4783}, // the AMD basket from E5's verification
		{1, 0},        // 0.1666… rounds down
		{3, 1},        // 0.5 rounds up — the half-up convention, pinned
		{6, 1},
	}
	for _, tc := range tests {
		if got := domain.ContainedVAT(tc.subtotal); got != tc.want {
			t.Errorf("ContainedVAT(%d) = %d, want %d", tc.subtotal, got, tc.want)
		}
	}
}

func TestShippingFor(t *testing.T) {
	free := func(n int64) *int64 { return &n }

	// The standard USD rate from migration 000017's bootstrap rows.
	rate := domain.ShippingRate{
		BaseMinor:               400,
		ColdChainSurchargeMinor: 600,
		FreeOverMinor:           free(7000),
	}

	tests := []struct {
		name         string
		subtotal     int64
		hasColdChain bool
		want         int64
	}{
		{"base only, under the threshold", 6200, false, 400},
		{"threshold reached exactly waives the base", 7000, false, 0},
		{"over the threshold", 9000, false, 0},
		// THE §1.4 READING: the surcharge survives free shipping. The mock's
		// own cart charges $6 chilled shipping on a subtotal past $70.
		{"cold chain under the threshold", 6200, true, 1000},
		{"cold chain over the threshold pays ONLY the surcharge", 9000, true, 600},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := rate.ShippingFor(tc.subtotal, tc.hasColdChain); got != tc.want {
				t.Errorf("ShippingFor(%d, %v) = %d, want %d",
					tc.subtotal, tc.hasColdChain, got, tc.want)
			}
		})
	}

	// A market with no threshold never ships free.
	noFree := domain.ShippingRate{BaseMinor: 1900}
	if got := noFree.ShippingFor(1_000_000, false); got != 1900 {
		t.Errorf("no-threshold rate waived the base: %d", got)
	}
}

func TestValidateAddress(t *testing.T) {
	complete := domain.Address{
		FirstName: "Anahit", LastName: "Sargsyan", Phone: "+374 91 000000",
		Street: "14 Abovyan St, apt 6", City: "Yerevan",
		PostalCode: "0009", Country: "AM",
	}
	if fields := domain.ValidateAddress(complete); len(fields) != 0 {
		t.Errorf("complete address rejected: %v", fields)
	}

	// Presence only — a five-digit code and an unusual phone must PASS,
	// because format rules written for one country reject real addresses in
	// another. What must fail is blank (including whitespace-only).
	odd := complete
	odd.PostalCode = "600001"
	odd.Phone = "010-555"
	if fields := domain.ValidateAddress(odd); len(fields) != 0 {
		t.Errorf("unusual formats rejected: %v", fields)
	}

	blank := domain.Address{FirstName: "  ", City: "Yerevan"}
	fields := domain.ValidateAddress(blank)
	for _, key := range []string{
		"address.first_name", "address.last_name", "address.phone",
		"address.street", "address.postal_code", "address.country",
	} {
		if fields[key] != domain.ValidationRequired {
			t.Errorf("missing %q in %v", key, fields)
		}
	}
	if _, ok := fields["address.city"]; ok {
		t.Error("filled city flagged")
	}
}

func TestValidatePayment(t *testing.T) {
	if f := domain.ValidatePayment(domain.PayCard, domain.CurrencyUSD); len(f) != 0 {
		t.Errorf("card in USD rejected: %v", f)
	}
	if f := domain.ValidatePayment(domain.PayCashOnDelivery, domain.CurrencyAMD); len(f) != 0 {
		t.Errorf("cash in AMD rejected: %v", f)
	}

	// The design's own words: "Cash — on delivery, AMD only".
	f := domain.ValidatePayment(domain.PayCashOnDelivery, domain.CurrencyUSD)
	if f["payment_method"] != domain.ValidationCashIsAMDOnly {
		t.Errorf("cash in USD: %v, want cash_is_amd_only", f)
	}

	f = domain.ValidatePayment("crypto", domain.CurrencyUSD)
	if f["payment_method"] != domain.ValidationInvalidPaymentMethod {
		t.Errorf("unknown method: %v", f)
	}
}

// F2: the payment machine, exhaustively — three states make nine ordered
// pairs, so the table can simply list all of them plus the garbage cases.
func TestValidPaymentTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"money arrives", domain.PaymentUnpaid, domain.PaymentPaid, true},
		{"money goes back", domain.PaymentPaid, domain.PaymentRefunded, true},

		{"no refunding money never taken", domain.PaymentUnpaid, domain.PaymentRefunded, false},
		{"no erasing a payment", domain.PaymentPaid, domain.PaymentUnpaid, false},
		{"refunded is terminal (to paid)", domain.PaymentRefunded, domain.PaymentPaid, false},
		{"refunded is terminal (to unpaid)", domain.PaymentRefunded, domain.PaymentUnpaid, false},
		{"no self transition (unpaid)", domain.PaymentUnpaid, domain.PaymentUnpaid, false},
		{"no self transition (paid)", domain.PaymentPaid, domain.PaymentPaid, false},
		{"no self transition (refunded)", domain.PaymentRefunded, domain.PaymentRefunded, false},
		{"unknown from-status", "garbage", domain.PaymentPaid, false},
		{"unknown to-status", domain.PaymentUnpaid, "garbage", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.ValidPaymentTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("ValidPaymentTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestValidPaymentStatus(t *testing.T) {
	for _, valid := range []string{domain.PaymentUnpaid, domain.PaymentPaid, domain.PaymentRefunded} {
		if !domain.ValidPaymentStatus(valid) {
			t.Errorf("ValidPaymentStatus(%q) = false, want true", valid)
		}
	}
	for _, invalid := range []string{"", "PAID", "card"} {
		if domain.ValidPaymentStatus(invalid) {
			t.Errorf("ValidPaymentStatus(%q) = true, want false", invalid)
		}
	}
}
