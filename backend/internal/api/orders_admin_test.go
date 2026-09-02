package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
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

// F2: POST /orders/{id}/cancel — the customer's self-service cancel. The
// window and ownership rules live in domain/store (their own tests); what
// the handler owes is the error VOCABULARY: 404 that hides existence from
// strangers, 409 too_late_to_cancel the client can translate, and a plain
// 200 with the cancelled order for the happy path.
func TestCancelMyOrder(t *testing.T) {
	newFake := func(status string) *fakeStore {
		fake := newFakeStore()
		fake.orders = []domain.Order{{
			ID: 12, UserID: 7, Status: status,
			Locale: domain.LocaleEN, Currency: domain.CurrencyAMD,
			PaymentMethod: domain.PayBankTransfer, PaymentStatus: domain.PaymentUnpaid,
		}}
		fake.usersByID[7] = domain.User{ID: 7, Email: "anahit@test.local", NotifyOrderUpdates: true}
		return fake
	}

	t.Run("anonymous gets 401", func(t *testing.T) {
		fake := newFake(domain.OrderPending)
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders/12/cancel", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("a stranger's attempt is a 404, not a 403", func(t *testing.T) {
		fake := newFake(domain.OrderPending)
		cookie := loginAs(fake, domain.User{ID: 5, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders/12/cancel", "", cookie)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want the existence-hiding 404", rec.Code)
		}
		if fake.orders[0].Status != domain.OrderPending {
			t.Errorf("a stranger changed the order to %q", fake.orders[0].Status)
		}
	})

	t.Run("the owner cancels a pending order", func(t *testing.T) {
		fake := newFake(domain.OrderPending)
		cookie := loginAs(fake, domain.User{ID: 7, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders/12/cancel", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		var got struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Status != "cancelled" {
			t.Errorf("status = %q, want cancelled", got.Status)
		}
	})

	t.Run("a confirmed order answers 409 too_late_to_cancel", func(t *testing.T) {
		fake := newFake(domain.OrderConfirmed)
		cookie := loginAs(fake, domain.User{ID: 7, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders/12/cancel", "", cookie)
		if rec.Code != http.StatusConflict {
			t.Fatalf("status = %d, want 409; body %s", rec.Code, rec.Body)
		}
		if !strings.Contains(rec.Body.String(), "too_late_to_cancel") {
			t.Errorf("body %s, want the too_late_to_cancel code the client translates", rec.Body)
		}
	})
}

// F2: the status-change mails. The fixture order's locale is ARMENIAN while
// every request in the test is plain English — which is exactly the point:
// the mail must follow the order's snapshot, not anyone's Accept-Language.
func TestUpdateOrderStatus_EmailsCustomer(t *testing.T) {
	newFake := func(notify bool) *fakeStore {
		fake := newFakeStore()
		fake.orders = []domain.Order{{
			ID: 12, UserID: 7, Status: domain.OrderPending,
			Locale: domain.LocaleHY, Currency: domain.CurrencyAMD,
			PaymentMethod: domain.PayBankTransfer, PaymentStatus: domain.PaymentUnpaid,
		}}
		fake.usersByID[7] = domain.User{
			ID: 7, Email: "anahit@test.local", NotifyOrderUpdates: notify,
		}
		return fake
	}

	patchStatus := func(fake *fakeStore, mailer *fakeMailer, body string) *httptest.ResponseRecorder {
		srv := newTestServerOpts(fake, api.Options{Mailer: mailer, PublicURL: "https://shop.example"})
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleAdmin})
		return doRequest(srv, http.MethodPatch, "/api/v1/admin/orders/12/status", body, cookie)
	}

	t.Run("confirming mails the customer in the order's language", func(t *testing.T) {
		fake, mailer := newFake(true), &fakeMailer{}
		if rec := patchStatus(fake, mailer, `{"status":"confirmed"}`); rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if len(mailer.sent) != 1 {
			t.Fatalf("%d mails sent, want 1", len(mailer.sent))
		}
		msg := mailer.sent[0]
		if msg.To != "anahit@test.local" {
			t.Errorf("to = %q, want the CUSTOMER, not the admin", msg.To)
		}
		if !strings.Contains(msg.Subject, "#12") || !strings.Contains(msg.Subject, "Պատվեր") {
			t.Errorf("subject = %q, want the Armenian confirmed subject", msg.Subject)
		}
		if !strings.Contains(msg.Text, "https://shop.example/hy/orders/12") {
			t.Errorf("body lacks the localized order link: %q", msg.Text)
		}
	})

	t.Run("the notify toggle off means the change happens silently", func(t *testing.T) {
		fake, mailer := newFake(false), &fakeMailer{}
		if rec := patchStatus(fake, mailer, `{"status":"confirmed"}`); rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if fake.orders[0].Status != domain.OrderConfirmed {
			t.Errorf("status = %q — the toggle must gate the MAIL, not the change", fake.orders[0].Status)
		}
		if len(mailer.sent) != 0 {
			t.Errorf("%d mails sent past an opted-out toggle", len(mailer.sent))
		}
	})

	t.Run("a refused transition sends nothing", func(t *testing.T) {
		fake, mailer := newFake(true), &fakeMailer{}
		if rec := patchStatus(fake, mailer, `{"status":"delivered"}`); rec.Code != http.StatusConflict {
			t.Fatalf("pending → delivered: status = %d, want 409", rec.Code)
		}
		if len(mailer.sent) != 0 {
			t.Errorf("%d mails sent for a transition that never happened", len(mailer.sent))
		}
	})

	t.Run("customer self-cancel sends the cancelled letter too", func(t *testing.T) {
		fake, mailer := newFake(true), &fakeMailer{}
		srv := newTestServerOpts(fake, api.Options{Mailer: mailer, PublicURL: "https://shop.example"})
		cookie := loginAs(fake, domain.User{ID: 7, Role: domain.RoleCustomer})
		rec := doRequest(srv, http.MethodPost, "/api/v1/orders/12/cancel", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if len(mailer.sent) != 1 || !strings.Contains(mailer.sent[0].Subject, "չեղարկված") {
			t.Errorf("sent = %+v, want one Armenian cancelled letter", mailer.sent)
		}
	})

	t.Run("payment flips never mail — only the parcel's machine writes letters", func(t *testing.T) {
		fake, mailer := newFake(true), &fakeMailer{}
		srv := newTestServerOpts(fake, api.Options{Mailer: mailer, PublicURL: "https://shop.example"})
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleAdmin})
		rec := doRequest(srv, http.MethodPatch, "/api/v1/admin/orders/12/payment", `{"payment_status":"paid"}`, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if len(mailer.sent) != 0 {
			t.Errorf("%d mails sent by a payment flip", len(mailer.sent))
		}
	})
}
