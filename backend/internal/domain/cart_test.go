package domain

import "testing"

func TestCartTotals(t *testing.T) {
	items := []CartItem{
		{PriceMinor: 650000, Qty: 3}, // 1_950_000
		{PriceMinor: 520000, Qty: 1}, //   520_000
	}

	if got := items[0].LineTotalMinor(); got != 1950000 {
		t.Errorf("LineTotalMinor = %d, want 1950000", got)
	}
	if got := CartTotalMinor(items); got != 2470000 {
		t.Errorf("CartTotalMinor = %d, want 2470000", got)
	}
	if got := CartTotalMinor(nil); got != 0 {
		t.Errorf("CartTotalMinor(nil) = %d, want 0", got)
	}
}
