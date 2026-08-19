package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// F2 (decision #94): the promo CRUD endpoints. Validation lives in domain
// (its own table test); what the handlers owe is the gate, the field-error
// envelope, the 409 vocabulary, and whole-value round-tripping.
func TestAdminPromoCRUD(t *testing.T) {
	adminCookie := func(fake *fakeStore) *http.Cookie {
		return loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
	}

	t.Run("the whole surface is admin-gated", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleCustomer})
		for _, req := range [][2]string{
			{http.MethodGet, "/api/v1/admin/promos"},
			{http.MethodPost, "/api/v1/admin/promos"},
			{http.MethodPut, "/api/v1/admin/promos/1"},
		} {
			if rec := doRequest(newTestServer(fake), req[0], req[1], "", nil); rec.Code != http.StatusUnauthorized {
				t.Errorf("%s %s anonymous: %d, want 401", req[0], req[1], rec.Code)
			}
			if rec := doRequest(newTestServer(fake), req[0], req[1], "", cookie); rec.Code != http.StatusForbidden {
				t.Errorf("%s %s customer: %d, want 403", req[0], req[1], rec.Code)
			}
		}
	})

	t.Run("create normalizes and echoes the stored code", func(t *testing.T) {
		fake := newFakeStore()
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/admin/promos",
			`{"code":"  honey15 ","kind":"percent","percent":15,"active":true,
			  "values":{"USD":{"min_subtotal_minor":2000}}}`, adminCookie(fake))
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		var got struct {
			Code    string `json:"code"`
			Percent *int   `json:"percent"`
			Values  map[string]struct {
				MinSubtotalMinor *int64 `json:"min_subtotal_minor"`
			} `json:"values"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Code != "HONEY15" {
			t.Errorf("code = %q, want the normalized HONEY15", got.Code)
		}
		if got.Percent == nil || *got.Percent != 15 {
			t.Errorf("percent = %v, want 15", got.Percent)
		}
		if v, ok := got.Values["USD"]; !ok || v.MinSubtotalMinor == nil || *v.MinSubtotalMinor != 2000 {
			t.Errorf("values did not round-trip: %s", rec.Body)
		}
	})

	t.Run("a case-variant duplicate is a 409 code_taken", func(t *testing.T) {
		fake := newFakeStore()
		fake.adminPromos = []domain.Promo{{ID: 1, Code: "HONEY15", Kind: domain.PromoFreeShipping}}
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/admin/promos",
			`{"code":"Honey15","kind":"free_shipping","active":true}`, adminCookie(fake))
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "code_taken") {
			t.Errorf("status = %d body %s, want 409 code_taken", rec.Code, rec.Body)
		}
	})

	t.Run("domain validation arrives as attachable field errors", func(t *testing.T) {
		fake := newFakeStore()
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/admin/promos",
			`{"code":"X","kind":"percent","active":true}`, adminCookie(fake))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		var got struct {
			Error struct {
				Fields map[string]string `json:"fields"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Error.Fields["percent"] != "required" {
			t.Errorf("fields = %v, want percent: required", got.Error.Fields)
		}
	})

	t.Run("update is whole-value and deactivation is just a field", func(t *testing.T) {
		fake := newFakeStore()
		fake.adminPromos = []domain.Promo{{ID: 1, Code: "HONEY15", Kind: domain.PromoPercent, Percent: 15, Active: true}}
		rec := doRequest(newTestServer(fake), http.MethodPut, "/api/v1/admin/promos/1",
			`{"code":"HONEY15","kind":"percent","percent":20,"active":false}`, adminCookie(fake))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if p := fake.adminPromos[0]; p.Percent != 20 || p.Active {
			t.Errorf("stored promo = %+v, want percent 20, inactive", p)
		}
	})

	t.Run("updating a ghost is a 404", func(t *testing.T) {
		fake := newFakeStore()
		rec := doRequest(newTestServer(fake), http.MethodPut, "/api/v1/admin/promos/99",
			`{"code":"GHOST","kind":"free_shipping","active":true}`, adminCookie(fake))
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404; body %s", rec.Code, rec.Body)
		}
	})

	t.Run("the list returns what was created", func(t *testing.T) {
		fake := newFakeStore()
		fake.adminPromos = []domain.Promo{
			{ID: 1, Code: "A", Kind: domain.PromoFreeShipping, Active: true},
			{ID: 2, Code: "B", Kind: domain.PromoPercent, Percent: 8},
		}
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/admin/promos", "", adminCookie(fake))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		var got []struct {
			Code    string `json:"code"`
			Percent *int   `json:"percent"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].Code != "A" || got[1].Percent == nil {
			t.Errorf("list = %+v", got)
		}
		// The free-shipping row must NOT carry a percent — the biconditional
		// on the wire.
		if got[0].Percent != nil {
			t.Errorf("free_shipping row carries percent %d", *got[0].Percent)
		}
	})
}
