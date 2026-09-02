package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// F2 (decision #96): the user administration endpoints. The count
// invariant lives in the store (its own suite, race included); the
// handlers owe the gate, the 400/404/409 vocabulary, and the response
// shape — including that password hashes never reach the wire.
func TestAdminUsers(t *testing.T) {
	seeded := func() *fakeStore {
		fake := newFakeStore()
		fake.usersByID[1] = domain.User{ID: 1, Email: "boss@test.local", Role: domain.RoleAdmin, PasswordHash: "secret-hash"}
		fake.usersByID[2] = domain.User{ID: 2, Email: "helper@test.local", Role: domain.RoleCustomer, FullName: "Anahit"}
		return fake
	}
	admin := func(fake *fakeStore) *http.Cookie {
		return loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
	}

	t.Run("both routes are admin-gated", func(t *testing.T) {
		fake := seeded()
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleCustomer})
		for _, req := range [][2]string{
			{http.MethodGet, "/api/v1/admin/users"},
			{http.MethodPatch, "/api/v1/admin/users/2/role"},
		} {
			if rec := doRequest(newTestServer(fake), req[0], req[1], "", nil); rec.Code != http.StatusUnauthorized {
				t.Errorf("%s %s anonymous: %d, want 401", req[0], req[1], rec.Code)
			}
			if rec := doRequest(newTestServer(fake), req[0], req[1], "", cookie); rec.Code != http.StatusForbidden {
				t.Errorf("%s %s customer: %d, want 403", req[0], req[1], rec.Code)
			}
		}
	})

	t.Run("the list shows roles and never a password hash", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/admin/users", "", admin(fake))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "secret-hash") {
			t.Fatal("a password hash reached the wire")
		}
		var got []struct {
			Email string `json:"email"`
			Role  string `json:"role"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].Email != "helper@test.local" || got[1].Role != "admin" {
			t.Errorf("list = %+v, want helper first (newest), boss as admin", got)
		}
	})

	t.Run("promote round-trips", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodPatch, "/api/v1/admin/users/2/role",
			`{"role":"admin"}`, admin(fake))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if fake.usersByID[2].Role != domain.RoleAdmin {
			t.Errorf("stored role = %q, want admin", fake.usersByID[2].Role)
		}
	})

	t.Run("demoting the only admin — self included — is a 409 last_admin", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodPatch, "/api/v1/admin/users/1/role",
			`{"role":"customer"}`, admin(fake))
		if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "last_admin") {
			t.Errorf("status = %d body %s, want 409 last_admin", rec.Code, rec.Body)
		}
		if fake.usersByID[1].Role != domain.RoleAdmin {
			t.Error("the refused demotion still changed the role")
		}
	})

	t.Run("an unknown role is a 400", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodPatch, "/api/v1/admin/users/2/role",
			`{"role":"superuser"}`, admin(fake))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("an unknown user is a 404", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodPatch, "/api/v1/admin/users/99/role",
			`{"role":"admin"}`, admin(fake))
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}
