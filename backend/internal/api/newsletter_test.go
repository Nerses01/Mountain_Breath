package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
)

func TestNewsletter(t *testing.T) {
	t.Run("subscribing mails a localized confirm link and answers 204", func(t *testing.T) {
		fake := newFakeStore()
		mailer := &fakeMailer{}
		srv := newTestServerOpts(fake, api.Options{Mailer: mailer, PublicURL: "https://shop.example"})

		rec := doRequest(srv, http.MethodPost, "/api/v1/newsletter/subscribe?lang=ru",
			`{"email": "Reader@Test.local"}`, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}

		if fake.newsletterEmail != "reader@test.local" {
			t.Errorf("stored email %q — normalization is the contract", fake.newsletterEmail)
		}
		if len(mailer.sent) != 1 {
			t.Fatalf("%d mails, want 1", len(mailer.sent))
		}
		wantLink := "https://shop.example/ru/newsletter/confirm/" + fake.newsletterToken
		if !strings.Contains(mailer.sent[0].Text, wantLink) {
			t.Errorf("mail %q missing link %q", mailer.sent[0].Text, wantLink)
		}
	})

	t.Run("an already-live subscriber gets 204 and NO mail", func(t *testing.T) {
		fake := newFakeStore()
		fake.newsletterLive = true
		mailer := &fakeMailer{}
		srv := newTestServerOpts(fake, api.Options{Mailer: mailer})

		rec := doRequest(srv, http.MethodPost, "/api/v1/newsletter/subscribe",
			`{"email": "reader@test.local"}`, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want the same 204 a stranger gets", rec.Code)
		}
		if len(mailer.sent) != 0 {
			t.Error("a live subscriber was re-mailed a confirm link — the spam loop")
		}
	})

	t.Run("a non-address is a field error before any row exists", func(t *testing.T) {
		fake := newFakeStore()
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/newsletter/subscribe",
			`{"email": "not-an-email"}`, nil)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if fake.newsletterEmail != "" {
			t.Error("an invalid address reached the store")
		}
	})

	t.Run("confirm posts the token back; an invented one is 400", func(t *testing.T) {
		fake := newFakeStore()
		fake.newsletterToken = "from-the-mail"
		srv := newTestServer(fake)

		rec := doRequest(srv, http.MethodPost, "/api/v1/newsletter/confirm",
			`{"token": "from-the-mail"}`, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("confirm = %d (%s)", rec.Code, rec.Body.String())
		}
		if fake.newsletterConfirmed != "from-the-mail" {
			t.Error("token did not reach the store")
		}

		rec = doRequest(srv, http.MethodPost, "/api/v1/newsletter/confirm",
			`{"token": "invented"}`, nil)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("invented token = %d, want 400", rec.Code)
		}
	})

	t.Run("unsubscribe is the same capability pointed the other way", func(t *testing.T) {
		fake := newFakeStore()
		fake.newsletterToken = "from-the-mail"
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/newsletter/unsubscribe",
			`{"token": "from-the-mail"}`, nil)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d", rec.Code)
		}
		if fake.newsletterUnsubscribed != "from-the-mail" {
			t.Error("token did not reach the store")
		}
	})
}
