package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// F2: PATCH /admin/orders/{id}/payment. The fake's UpdateOrderPaymentStatus
// runs the real domain machine, so this walks the endpoint's whole surface:
// who may call it, the three error shapes (400 not-a-status, 409 refused
// transition, 404 no order), and the two legal flips in sequence.
func TestUpdateOrderPayment(t *testing.T) {
	newFake := func() *fakeStore {
		fake := newFakeStore()
		fake.orders = []domain.Order{{
			ID: 12, UserID: 1, Status: domain.OrderPending,
			Currency: domain.CurrencyAMD, TotalMinor: 640000,
			PaymentMethod: domain.PayBankTransfer, PaymentStatus: domain.PaymentUnpaid,
		}}
		return fake
	}

	patch := func(fake *fakeStore, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		return doRequest(newTestServer(fake), http.MethodPatch, path, body, cookie)
	}

	t.Run("anonymous gets 401, customer 403", func(t *testing.T) {
		fake := newFake()
		if rec := patch(fake, "/api/v1/admin/orders/12/payment", `{"payment_status":"paid"}`, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("anonymous: status = %d, want 401", rec.Code)
		}
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})
		if rec := patch(fake, "/api/v1/admin/orders/12/payment", `{"payment_status":"paid"}`, cookie); rec.Code != http.StatusForbidden {
			t.Errorf("customer: status = %d, want 403", rec.Code)
		}
		// And through neither door did the order change.
		if fake.orders[0].PaymentStatus != domain.PaymentUnpaid {
			t.Errorf("payment flipped to %q by a refused request", fake.orders[0].PaymentStatus)
		}
	})

	t.Run("a word that is not a payment status is a 400", func(t *testing.T) {
		fake := newFake()
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleAdmin})
		rec := patch(fake, "/api/v1/admin/orders/12/payment", `{"payment_status":"confirmed"}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400; body %s", rec.Code, rec.Body)
		}
	})

	t.Run("a real status the machine refuses is a 409", func(t *testing.T) {
		fake := newFake()
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleAdmin})
		rec := patch(fake, "/api/v1/admin/orders/12/payment", `{"payment_status":"refunded"}`, cookie)
		if rec.Code != http.StatusConflict {
			t.Errorf("unpaid → refunded: status = %d, want 409; body %s", rec.Code, rec.Body)
		}
	})

	t.Run("unknown order is a 404", func(t *testing.T) {
		fake := newFake()
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleAdmin})
		rec := patch(fake, "/api/v1/admin/orders/99/payment", `{"payment_status":"paid"}`, cookie)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404; body %s", rec.Code, rec.Body)
		}
	})

	t.Run("mark paid, then refund — the legal path end to end", func(t *testing.T) {
		fake := newFake()
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleAdmin})

		rec := patch(fake, "/api/v1/admin/orders/12/payment", `{"payment_status":"paid"}`, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("mark paid: status = %d; body %s", rec.Code, rec.Body)
		}
		var got struct {
			Status        string `json:"status"`
			PaymentStatus string `json:"payment_status"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.PaymentStatus != "paid" {
			t.Errorf("payment_status = %q, want paid", got.PaymentStatus)
		}
		// Orthogonality on the wire: the parcel's state is untouched.
		if got.Status != "pending" {
			t.Errorf("order status = %q, want pending", got.Status)
		}

		rec = patch(fake, "/api/v1/admin/orders/12/payment", `{"payment_status":"refunded"}`, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("refund: status = %d; body %s", rec.Code, rec.Body)
		}
		if fake.orders[0].PaymentStatus != domain.PaymentRefunded {
			t.Errorf("stored payment = %q, want refunded", fake.orders[0].PaymentStatus)
		}
	})
}
