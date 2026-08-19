package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// F2 (decision #97): GET /account/data. The export composes existing
// store reads; what the handler owes is completeness of the shape and
// that nothing secret rides along.
func TestAccountData(t *testing.T) {
	fake := newFakeStore()
	fake.usersByID[7] = domain.User{
		ID: 7, Email: "anahit@test.local", Role: domain.RoleCustomer,
		FullName: "Anahit", PasswordHash: "secret-hash",
		NotifyOrderUpdates: true, CreatedAt: time.Now(),
	}
	fake.orders = []domain.Order{{
		ID: 12, UserID: 7, Status: domain.OrderDelivered,
		Currency: domain.CurrencyAMD, TotalMinor: 640000,
		PaymentMethod: domain.PayBankTransfer, PaymentStatus: domain.PaymentPaid,
	}}
	fake.userReviews = []domain.Review{{Rating: 5, Title: "Wonderful", Status: domain.ReviewPending}}
	fake.userReviewSlugs = []string{"wild-honey"}

	cookie := loginAs(fake, domain.User{ID: 7, Role: domain.RoleCustomer})

	t.Run("anonymous gets 401", func(t *testing.T) {
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/account/data", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("the export carries every promised section and no secrets", func(t *testing.T) {
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/account/data", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if strings.Contains(rec.Body.String(), "secret-hash") {
			t.Fatal("the password hash reached the export")
		}
		var got struct {
			Account struct {
				Email string `json:"email"`
			} `json:"account"`
			Orders []struct {
				ID int64 `json:"id"`
			} `json:"orders"`
			Reviews []struct {
				ProductSlug string `json:"product_slug"`
				Status      string `json:"status"`
			} `json:"reviews"`
			Newsletter     string `json:"newsletter"`
			ActiveSessions int    `json:"active_sessions"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Account.Email != "anahit@test.local" {
			t.Errorf("account.email = %q", got.Account.Email)
		}
		if len(got.Orders) != 1 || got.Orders[0].ID != 12 {
			t.Errorf("orders = %+v", got.Orders)
		}
		// Pending reviews are still the person's words — present.
		if len(got.Reviews) != 1 || got.Reviews[0].ProductSlug != "wild-honey" || got.Reviews[0].Status != "pending" {
			t.Errorf("reviews = %+v", got.Reviews)
		}
		// loginAs put exactly one live session in the fake's table.
		if got.ActiveSessions != 1 {
			t.Errorf("active_sessions = %d, want 1", got.ActiveSessions)
		}
	})
}

// F2 (decision #97): DELETE /account — re-authentication rules and the
// error vocabulary. The graph semantics live in the store's suite.
func TestDeleteAccount(t *testing.T) {
	hash := func(pw string) string {
		h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		return string(h)
	}

	t.Run("a password account must present its password", func(t *testing.T) {
		fake := newFakeStore()
		fake.usersByID[7] = domain.User{ID: 7, Email: "a@test.local", Role: domain.RoleCustomer, PasswordHash: hash("hunter2!")}
		cookie := loginAs(fake, domain.User{ID: 7, Role: domain.RoleCustomer, PasswordHash: hash("hunter2!")})

		rec := doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/account",
			`{"current_password":"wrong"}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("wrong password: status = %d, want 400; body %s", rec.Code, rec.Body)
		}
		if _, ok := fake.usersByID[7]; !ok {
			t.Fatal("a wrong password still deleted the account")
		}

		rec = doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/account",
			`{"current_password":"hunter2!"}`, cookie)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("right password: status = %d; body %s", rec.Code, rec.Body)
		}
		if _, ok := fake.usersByID[7]; ok {
			t.Error("the account survived its deletion")
		}
		// The response must kill the browser's cookie.
		found := false
		for _, c := range rec.Result().Cookies() {
			if c.Name == "mb_session" && c.MaxAge < 0 {
				found = true
			}
		}
		if !found {
			t.Error("no expiring session cookie on the delete response")
		}
	})

	t.Run("an OAuth-only account (empty hash) deletes without a password", func(t *testing.T) {
		fake := newFakeStore()
		fake.usersByID[7] = domain.User{ID: 7, Email: "g@test.local", Role: domain.RoleCustomer, PasswordHash: ""}
		cookie := loginAs(fake, domain.User{ID: 7, Role: domain.RoleCustomer, PasswordHash: ""})

		rec := doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/account", `{}`, cookie)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
	})

	t.Run("the last admin cannot delete themself", func(t *testing.T) {
		fake := newFakeStore()
		fake.usersByID[1] = domain.User{ID: 1, Email: "boss@test.local", Role: domain.RoleAdmin, PasswordHash: ""}
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin, PasswordHash: ""})

		rec := doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/account", `{}`, cookie)
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "last_admin") {
			t.Errorf("status = %d body %s, want 409 last_admin", rec.Code, rec.Body)
		}
		if _, ok := fake.usersByID[1]; !ok {
			t.Error("the refused deletion still removed the account")
		}
	})
}
