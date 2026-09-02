package domain

import "testing"

// Table-driven test: the Go idiom. One slice of cases, one loop, and
// `go test` reports each case by name via t.Run.
func TestValidOrderTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"pending to confirmed", OrderPending, OrderConfirmed, true},
		{"pending to cancelled", OrderPending, OrderCancelled, true},
		{"confirmed to shipped", OrderConfirmed, OrderShipped, true},
		{"confirmed to cancelled", OrderConfirmed, OrderCancelled, true},
		{"shipped to delivered", OrderShipped, OrderDelivered, true},

		{"pending straight to shipped", OrderPending, OrderShipped, false},
		{"pending straight to delivered", OrderPending, OrderDelivered, false},
		{"shipped to cancelled", OrderShipped, OrderCancelled, false},
		{"delivered is terminal", OrderDelivered, OrderCancelled, false},
		{"cancelled is terminal", OrderCancelled, OrderPending, false},
		{"no self transition", OrderPending, OrderPending, false},
		{"unknown from-status", "garbage", OrderConfirmed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidOrderTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("ValidOrderTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

// F2: the customer's window is PENDING only — narrower than the machine,
// which also allows confirmed → cancelled for the admin. The test walks
// every status so a new state cannot silently widen the window.
func TestCustomerMayCancelOrder(t *testing.T) {
	for status, want := range map[string]bool{
		OrderPending:   true,
		OrderConfirmed: false,
		OrderShipped:   false,
		OrderDelivered: false,
		OrderCancelled: false,
		"garbage":      false,
	} {
		if got := CustomerMayCancelOrder(status); got != want {
			t.Errorf("CustomerMayCancelOrder(%q) = %v, want %v", status, got, want)
		}
	}
}

func TestValidOrderStatus(t *testing.T) {
	for _, valid := range []string{OrderPending, OrderConfirmed, OrderShipped, OrderDelivered, OrderCancelled} {
		if !ValidOrderStatus(valid) {
			t.Errorf("ValidOrderStatus(%q) = false, want true", valid)
		}
	}
	for _, invalid := range []string{"", "PENDING", "unknown"} {
		if ValidOrderStatus(invalid) {
			t.Errorf("ValidOrderStatus(%q) = true, want false", invalid)
		}
	}
}
