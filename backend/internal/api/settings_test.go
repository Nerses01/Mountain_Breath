package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// A5's handler contracts: profile writes validate and come back as the
// fresh user; the password change verifies the CURRENT password before
// anything, keeps the caller's own session, and speaks in field errors;
// the notifications panel reads both real channels in one response.

func TestUpdateProfileHandler(t *testing.T) {
	t.Run("valid fields are stored and echoed back", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Email: "a@x.am", Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodPatch, "/api/v1/account/profile",
			`{"full_name": "Anahit Sargsyan", "phone": "+374 91 000000"}`, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		if fake.profileName != "Anahit Sargsyan" || fake.profilePhone != "+374 91 000000" {
			t.Errorf("store got %q / %q", fake.profileName, fake.profilePhone)
		}
		var got struct {
			FullName string `json:"full_name"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.FullName != "Anahit Sargsyan" {
			t.Errorf("response full_name = %q — the client sets its cache from this", got.FullName)
		}
	})

	t.Run("an oversized name is a field error", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		long := make([]byte, 121)
		for i := range long {
			long[i] = 'a'
		}
		rec := doRequest(newTestServer(fake), http.MethodPatch, "/api/v1/account/profile",
			`{"full_name": "`+string(long)+`", "phone": ""}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

func TestChangePasswordHandler(t *testing.T) {
	user := domain.User{
		ID: 1, Email: "a@x.am", Role: domain.RoleCustomer,
		PasswordHash: "", // set per test
	}

	t.Run("wrong current password is a FIELD error, not a 401", func(t *testing.T) {
		fake := newFakeStore()
		u := user
		u.PasswordHash = testPasswordHash(t, "the-real-one")
		cookie := loginAs(fake, u)

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/account/password",
			`{"current_password": "not-the-one", "new_password": "long-enough-pass"}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400 (%s)", rec.Code, rec.Body.String())
		}
		var got struct {
			Error struct {
				Fields map[string]string `json:"fields"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Error.Fields["current_password"] != "incorrect_password" {
			t.Errorf("fields = %v", got.Error.Fields)
		}
		if fake.newHash != "" {
			t.Error("the store was called despite the wrong password")
		}
	})

	t.Run("a short new password never reaches the bcrypt compare", func(t *testing.T) {
		fake := newFakeStore()
		u := user
		u.PasswordHash = testPasswordHash(t, "the-real-one")
		cookie := loginAs(fake, u)

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/account/password",
			`{"current_password": "the-real-one", "new_password": "short"}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("success keeps the caller's own session", func(t *testing.T) {
		fake := newFakeStore()
		u := user
		u.PasswordHash = testPasswordHash(t, "the-real-one")
		cookie := loginAs(fake, u)

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/account/password",
			`{"current_password": "the-real-one", "new_password": "long-enough-pass"}`, cookie)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (%s)", rec.Code, rec.Body.String())
		}
		if fake.newHash == "" {
			t.Error("no new hash reached the store")
		}
		// The kept token is the cookie that authenticated this request.
		if fake.keptToken != cookie.Value {
			t.Errorf("kept token = %q, want the caller's own %q", fake.keptToken, cookie.Value)
		}
	})

	t.Run("an OAuth account (empty hash) fails closed", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, user) // PasswordHash "" — bcrypt matches nothing

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/account/password",
			`{"current_password": "", "new_password": "long-enough-pass"}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

func TestNotificationsHandlers(t *testing.T) {
	t.Run("the panel reads both real channels at once", func(t *testing.T) {
		fake := newFakeStore()
		fake.newsletterStatus = domain.NewsletterSubscribed
		cookie := loginAs(fake, domain.User{ID: 1, Email: "a@x.am", Role: domain.RoleCustomer, NotifyOrderUpdates: true})

		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/account/notifications", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		var got struct {
			OrderUpdates bool   `json:"order_updates"`
			Newsletter   string `json:"newsletter"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if !got.OrderUpdates || got.Newsletter != "subscribed" {
			t.Errorf("got %+v", got)
		}
	})

	t.Run("the order-updates toggle writes through", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodPatch, "/api/v1/account/notifications",
			`{"order_updates": false}`, cookie)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
		if fake.notifyOrderUpdates == nil || *fake.notifyOrderUpdates {
			t.Error("the store did not receive order_updates=false")
		}
	})

	t.Run("the account unsubscribe uses the session's email", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Email: "a@x.am", Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/account/newsletter", "", cookie)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
		if fake.unsubscribedEmail != "a@x.am" {
			t.Errorf("unsubscribed %q", fake.unsubscribedEmail)
		}
	})
}
