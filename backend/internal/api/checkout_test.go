package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

const validCheckoutBody = `{
	"address": {
		"first_name": "Anahit", "last_name": "Sargsyan",
		"phone": "+374 91 000000", "street": "14 Abovyan St, apt 6",
		"city": "Yerevan", "postal_code": "0009", "country": "AM"
	},
	"payment_method": "card",
	"delivery_note": "Ring twice",
	"leave_with_neighbour": true
}`

func cartWithOneItem() []domain.CartItem {
	return []domain.CartItem{{
		VariantID: 1, Qty: 1, PriceMinor: 1400,
		Prices: domain.Money{domain.CurrencyUSD: 1400, domain.CurrencyAMD: 6700},
	}}
}

func TestCheckout(t *testing.T) {
	t.Run("a complete checkout reaches the store intact", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders", validCheckoutBody, cookie)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (%s)", rec.Code, rec.Body.String())
		}

		in := fake.lastCheckout
		if in.Address.City != "Yerevan" || in.PaymentMethod != domain.PayCard ||
			in.DeliveryNote != "Ring twice" || !in.LeaveWithNeighbour {
			t.Errorf("checkout input mangled in transit: %+v", in)
		}
	})

	t.Run("missing fields come back as one 400 with JSON paths", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		body := `{"address": {"first_name": "A"}, "payment_method": "card"}`
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders", body, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
		}

		var envelope struct {
			Error struct {
				Fields map[string]string `json:"fields"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		// The keys are the JSON paths the form posts, so the React form can
		// attach each error to its input — every gap reported in ONE round
		// trip, not one per submit.
		for _, key := range []string{
			"address.last_name", "address.phone", "address.street",
			"address.city", "address.postal_code", "address.country",
		} {
			if envelope.Error.Fields[key] != "required" {
				t.Errorf("missing %q in %v", key, envelope.Error.Fields)
			}
		}
	})

	// THE test the plan asked for by name: the server owns every number. The
	// request shape has no money field, and DisallowUnknownFields makes a
	// body that smuggles one a 400 — the total is not merely ignored, the
	// request is refused before a handler line runs.
	t.Run("a client-supplied total is refused outright", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		body := `{
			"address": {
				"first_name": "Anahit", "last_name": "Sargsyan",
				"phone": "+374 91 000000", "street": "14 Abovyan St",
				"city": "Yerevan", "postal_code": "0009", "country": "AM"
			},
			"payment_method": "card",
			"total_minor": 1
		}`
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders", body, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
		}
		if fake.lastCheckout.PaymentMethod != "" {
			t.Error("the store was reached despite the smuggled total")
		}
	})

	t.Run("cash on delivery in dollars is a field error", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		body := `{
			"address": {
				"first_name": "Anahit", "last_name": "Sargsyan",
				"phone": "+374 91 000000", "street": "14 Abovyan St",
				"city": "Yerevan", "postal_code": "0009", "country": "AM"
			},
			"payment_method": "cash_on_delivery"
		}`
		// No ?currency= and no cookie: the request resolves to USD, and the
		// design's own words apply — "Cash: on delivery, AMD only".
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders", body, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
		}

		var envelope struct {
			Error struct {
				Fields map[string]string `json:"fields"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error.Fields["payment_method"] != "cash_is_amd_only" {
			t.Errorf("fields = %v, want payment_method: cash_is_amd_only", envelope.Error.Fields)
		}
	})

	t.Run("anonymous gets 401", func(t *testing.T) {
		rec := doRequest(newTestServer(newFakeStore()), http.MethodPost, "/api/v1/orders", validCheckoutBody, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})
}

func TestGetOrder(t *testing.T) {
	fake := newFakeStore()
	fake.orders = []domain.Order{{
		ID: 12, UserID: 1, Status: domain.OrderPending, Currency: domain.CurrencyUSD,
		TotalMinor: 6400,
		Totals: domain.OrderTotals{
			SubtotalMinor: 6200, ShippingMinor: 600, DiscountMinor: 400,
			TaxMinor: 1033, TotalMinor: 6400,
		},
		PaymentMethod: domain.PayCard, PaymentStatus: domain.PaymentUnpaid,
		ShipTo: &domain.Address{FirstName: "Anahit", Street: "14 Abovyan St", City: "Yerevan"},
	}}

	t.Run("the owner sees the full breakdown", func(t *testing.T) {
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/orders/12", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}

		var got struct {
			SubtotalMinor int64 `json:"subtotal_minor"`
			ShippingMinor int64 `json:"shipping_minor"`
			DiscountMinor int64 `json:"discount_minor"`
			TaxMinor      int64 `json:"tax_minor"`
			TotalMinor    int64 `json:"total_minor"`
			ShipTo        *struct {
				City string `json:"city"`
			} `json:"ship_to"`
			PaymentStatus string `json:"payment_status"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.SubtotalMinor+got.ShippingMinor-got.DiscountMinor != got.TotalMinor {
			t.Errorf("the response's own figures do not balance: %+v", got)
		}
		if got.ShipTo == nil || got.ShipTo.City != "Yerevan" {
			t.Errorf("address snapshot missing: %+v", got.ShipTo)
		}
		if got.PaymentStatus != "unpaid" {
			t.Errorf("payment_status = %q, want unpaid", got.PaymentStatus)
		}
	})

	// 404, not 403: an order is private data, and a 403 would confirm to an
	// id-enumerator that order 12 exists and is somebody's. Compare E4,
	// where a non-purchaser reviewing a PUBLIC product rightly got 403.
	t.Run("someone else's order does not exist, as far as you know", func(t *testing.T) {
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/orders/12", "", cookie)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("the admin sees any order", func(t *testing.T) {
		cookie := loginAs(fake, domain.User{ID: 3, Role: domain.RoleAdmin})
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/orders/12", "", cookie)
		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})
}

func TestGetDefaultAddress(t *testing.T) {
	t.Run("a saved address comes back for pre-filling", func(t *testing.T) {
		fake := newFakeStore()
		fake.defaultAddress = &domain.AddressEntry{
			LeaveWithNeighbour: true,
			Address:            domain.Address{FirstName: "Anahit", City: "Yerevan", Country: "AM"},
		}
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/account/address", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		var got struct {
			City               string `json:"city"`
			LeaveWithNeighbour bool   `json:"leave_with_neighbour"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.City != "Yerevan" {
			t.Errorf("city = %q", got.City)
		}
		// A4: the neighbour prefill rides the same response (log #88).
		if !got.LeaveWithNeighbour {
			t.Error("leave_with_neighbour missing from the prefill response")
		}
	})

	t.Run("a first-time customer gets 404, which the form renders empty", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/account/address", "", cookie)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}
