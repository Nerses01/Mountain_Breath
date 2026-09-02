package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/mail"
)

// fakeMailer records instead of delivering — the handler tests assert on
// what WOULD have been sent, which is the handler's whole observable job.
type fakeMailer struct {
	sent []mail.Message
}

func (f *fakeMailer) Send(_ context.Context, m mail.Message) error {
	f.sent = append(f.sent, m)
	return nil
}

// ── Password reset ────────────────────────────────────────────────────────

func TestForgotPassword(t *testing.T) {
	t.Run("a known email gets a localized link; the response is 204", func(t *testing.T) {
		fake := newFakeStore()
		fake.userByEmail = &domain.User{ID: 7, Email: "anahit@test.local"}
		mailer := &fakeMailer{}
		srv := newTestServerOpts(fake, api.Options{Mailer: mailer, PublicURL: "https://shop.example"})

		rec := doRequest(srv, http.MethodPost, "/api/v1/auth/forgot-password?lang=hy",
			`{"email": "Anahit@Test.local"}`, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204 (%s)", rec.Code, rec.Body.String())
		}

		if len(mailer.sent) != 1 {
			t.Fatalf("%d mails sent, want 1", len(mailer.sent))
		}
		m := mailer.sent[0]
		if m.To != "anahit@test.local" {
			t.Errorf("mail to %q", m.To)
		}
		// The link lands in the requester's language, on the PUBLIC origin,
		// and carries the raw token the store was given — the token in the
		// inbox and the hash in the table must be the same secret.
		wantLink := "https://shop.example/hy/reset-password/" + fake.resetToken
		if !strings.Contains(m.Text, wantLink) {
			t.Errorf("mail text %q missing link %q", m.Text, wantLink)
		}
		if fake.resetUserID != 7 || fake.resetToken == "" {
			t.Errorf("token not stored for the user: id=%d token=%q", fake.resetUserID, fake.resetToken)
		}
	})

	t.Run("an unknown email answers identically — no enumeration oracle", func(t *testing.T) {
		fake := newFakeStore() // GetUserByEmail → ErrNotFound
		mailer := &fakeMailer{}
		srv := newTestServerOpts(fake, api.Options{Mailer: mailer})

		rec := doRequest(srv, http.MethodPost, "/api/v1/auth/forgot-password",
			`{"email": "stranger@test.local"}`, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want the same 204 a member gets", rec.Code)
		}
		if len(mailer.sent) != 0 {
			t.Error("a mail was sent for a nonexistent account")
		}
	})
}

func TestResetPassword(t *testing.T) {
	t.Run("a valid token sets the new hash", func(t *testing.T) {
		fake := newFakeStore()
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/auth/reset-password",
			`{"token": "raw-token", "password": "brand-new-password"}`, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		if fake.consumedToken != "raw-token" {
			t.Errorf("consumed %q", fake.consumedToken)
		}
		if fake.newPasswordHash == "" || fake.newPasswordHash == "brand-new-password" {
			t.Error("the password reached the store unhashed (or not at all)")
		}
	})

	t.Run("spent, expired and invented tokens all read the same", func(t *testing.T) {
		fake := newFakeStore()
		fake.consumeErr = domain.ErrNotFound
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/auth/reset-password",
			`{"token": "whatever", "password": "brand-new-password"}`, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		var envelope struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Error.Code != "invalid_token" {
			t.Errorf("code = %q — one answer for every kind of no", envelope.Error.Code)
		}
	})

	t.Run("a short password is refused before any token is spent", func(t *testing.T) {
		fake := newFakeStore()
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/auth/reset-password",
			`{"token": "raw-token", "password": "short"}`, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if fake.consumedToken != "" {
			t.Error("token spent on a submission that failed validation")
		}
	})
}

// ── Rate limiting & remember-me ───────────────────────────────────────────

func TestLoginRateLimit(t *testing.T) {
	fake := newFakeStore() // unknown email → every attempt fails
	srv := newTestServer(fake)

	var last int
	for i := 0; i < 11; i++ {
		rec := doRequest(srv, http.MethodPost, "/api/v1/auth/login",
			`{"email": "target@test.local", "password": "guess"}`, nil)
		last = rec.Code
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("11th attempt = %d, want 429", last)
	}

	// Another account from the same IP still gets through — the key is
	// (ip, email), so hammering one address does not lock out the shop.
	rec := doRequest(srv, http.MethodPost, "/api/v1/auth/login",
		`{"email": "other@test.local", "password": "guess"}`, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("different email = %d, want a plain 401", rec.Code)
	}
}

func TestLoginRememberMe(t *testing.T) {
	cookieFor := func(body string) *http.Cookie {
		fake := newFakeStore()
		fake.userByEmail = &domain.User{ID: 1, Email: "anahit@test.local",
			PasswordHash: testPasswordHash(t, "correct-horse-battery")}
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/auth/login", body, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("login = %d (%s)", rec.Code, rec.Body.String())
		}
		for _, c := range rec.Result().Cookies() {
			if c.Name == "mb_session" {
				return c
			}
		}
		t.Fatal("no session cookie set")
		return nil
	}

	short := cookieFor(`{"email": "anahit@test.local", "password": "correct-horse-battery"}`)
	long := cookieFor(`{"email": "anahit@test.local", "password": "correct-horse-battery", "remember": true}`)

	// A week vs a month — the checkbox is the difference, and the cookie's
	// MaxAge is where the choice becomes visible to a test.
	if short.MaxAge != 7*24*3600 {
		t.Errorf("default session MaxAge = %d, want a week", short.MaxAge)
	}
	if long.MaxAge != 30*24*3600 {
		t.Errorf("remembered session MaxAge = %d, want thirty days", long.MaxAge)
	}
}

// ── Wishlist ──────────────────────────────────────────────────────────────

func TestWishlist(t *testing.T) {
	fake := newFakeStore()
	fake.products = []domain.Product{{ID: 5, Slug: "honey", Name: "Honey"}}
	fake.cart = []domain.CartItem{{VariantID: 9, Qty: 2}}
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})
	srv := newTestServer(fake)

	t.Run("heart, twice, is one fact", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			rec := doRequest(srv, http.MethodPut, "/api/v1/wishlist/5", "", cookie)
			if rec.Code != http.StatusNoContent {
				t.Fatalf("status = %d", rec.Code)
			}
		}
		if !fake.wishlist[5] {
			t.Error("product not hearted")
		}
	})

	t.Run("hearting a ghost is 404", func(t *testing.T) {
		rec := doRequest(srv, http.MethodPut, "/api/v1/wishlist/99", "", cookie)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("save-for-later moves the line", func(t *testing.T) {
		rec := doRequest(srv, http.MethodPost, "/api/v1/wishlist/save-for-later",
			`{"variant_id": 9}`, cookie)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		if fake.savedForLater != 9 {
			t.Errorf("saved variant = %d", fake.savedForLater)
		}
	})

	t.Run("anonymous hearts get 401", func(t *testing.T) {
		rec := doRequest(srv, http.MethodPut, "/api/v1/wishlist/5", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})
}

// ── Address book ──────────────────────────────────────────────────────────

func TestAddressBook(t *testing.T) {
	fake := newFakeStore()
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})
	srv := newTestServer(fake)

	body := `{
		"label": "Home", "is_default": false,
		"first_name": "Anahit", "last_name": "Sargsyan",
		"phone": "+374 91 000000", "street": "14 Abovyan St",
		"city": "Yerevan", "postal_code": "0009", "country": "AM"
	}`

	t.Run("create validates with the checkout's field keys", func(t *testing.T) {
		rec := doRequest(srv, http.MethodPost, "/api/v1/account/addresses",
			`{"label": "Home", "first_name": "A", "last_name": "", "phone": "", "street": "", "city": "", "postal_code": "", "country": "", "is_default": false}`,
			cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "address.street") {
			t.Errorf("errors do not use the checkout's keys: %s", rec.Body.String())
		}
	})

	t.Run("the first entry becomes the default regardless", func(t *testing.T) {
		rec := doRequest(srv, http.MethodPost, "/api/v1/account/addresses", body, cookie)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		var got struct {
			ID        int64 `json:"id"`
			IsDefault bool  `json:"is_default"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if !got.IsDefault {
			t.Error("first address did not become the default")
		}
	})

	t.Run("someone else's address is 404", func(t *testing.T) {
		rec := doRequest(srv, http.MethodDelete, "/api/v1/account/addresses/99", "", cookie)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

// ── Google OAuth, against a fake Google ───────────────────────────────────

func TestGoogleOAuth(t *testing.T) {
	// A pretend Google: hands out a token for any code, answers userinfo
	// with a fixed verified identity.
	google := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if r.FormValue("code") != "good-code" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "at-123"})
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer at-123" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub": "g-sub-1", "email": "Anahit@Gmail.example", "email_verified": true,
			})
		}
	}))
	defer google.Close()

	opts := api.Options{
		PublicURL:          "https://shop.example",
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleAuthURL:      google.URL + "/auth",
		GoogleTokenURL:     google.URL + "/token",
		GoogleUserinfoURL:  google.URL + "/userinfo",
	}

	start := func(t *testing.T, srv http.Handler, lang string) (state string, cookie *http.Cookie) {
		t.Helper()
		rec := doRequest(srv, http.MethodGet, "/api/v1/auth/oauth/google"+lang, "", nil)
		if rec.Code != http.StatusFound {
			t.Fatalf("start = %d, want 302", rec.Code)
		}
		loc, err := rec.Result().Location()
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range rec.Result().Cookies() {
			if c.Name == "mb_oauth_state" {
				cookie = c
			}
		}
		if cookie == nil {
			t.Fatal("no state cookie set")
		}
		return loc.Query().Get("state"), cookie
	}

	t.Run("the whole flow mints a session and links the identity", func(t *testing.T) {
		fake := newFakeStore()
		srv := newTestServerOpts(fake, opts)
		state, cookie := start(t, srv, "?lang=hy")

		rec := doRequest(srv, http.MethodGet,
			"/api/v1/auth/oauth/google/callback?code=good-code&state="+state, "", cookie)
		if rec.Code != http.StatusFound {
			t.Fatalf("callback = %d (%s)", rec.Code, rec.Body.String())
		}
		loc, _ := rec.Result().Location()
		// Back to the public origin, in the language the flow started from.
		if loc.String() != "https://shop.example/hy/" {
			t.Errorf("redirected to %s", loc)
		}
		if fake.oauthProvider != "google" || fake.oauthSubject != "g-sub-1" {
			t.Errorf("identity = %s/%s", fake.oauthProvider, fake.oauthSubject)
		}
		// The provider's email arrives NORMALIZED, like every email here.
		if fake.oauthEmail != "anahit@gmail.example" {
			t.Errorf("email = %q", fake.oauthEmail)
		}
		var sessionSet bool
		for _, c := range rec.Result().Cookies() {
			if c.Name == "mb_session" && c.Value != "" {
				sessionSet = true
			}
		}
		if !sessionSet {
			t.Error("no session cookie after a successful flow")
		}
	})

	t.Run("a forged state gets no session — the CSRF case", func(t *testing.T) {
		fake := newFakeStore()
		srv := newTestServerOpts(fake, opts)
		_, cookie := start(t, srv, "")

		rec := doRequest(srv, http.MethodGet,
			"/api/v1/auth/oauth/google/callback?code=good-code&state=forged", "", cookie)
		if rec.Code != http.StatusFound {
			t.Fatalf("callback = %d", rec.Code)
		}
		loc, _ := rec.Result().Location()
		if !strings.Contains(loc.String(), "oauth_error=1") {
			t.Errorf("redirected to %s, want the login page with the error flag", loc)
		}
		if fake.oauthProvider != "" {
			t.Error("the store was asked to resolve a forged flow")
		}
	})

	t.Run("unconfigured Google is a 404, not a broken redirect", func(t *testing.T) {
		rec := doRequest(newTestServer(newFakeStore()), http.MethodGet, "/api/v1/auth/oauth/google", "", nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}
