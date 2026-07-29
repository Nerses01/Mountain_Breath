package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

func TestHealth(t *testing.T) {
	rec := doRequest(newTestServer(newFakeStore()), http.MethodGet, "/health", "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestListCategories(t *testing.T) {
	fake := newFakeStore()
	fake.categories = []domain.Category{
		{ID: 1, Slug: "tea", Name: "Tea"},
		{ID: 2, Slug: "coffee", Name: "Coffee"},
	}

	rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/categories", "", nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not a JSON array: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d categories, want 2", len(got))
	}
}

// The admin gate, exercised through the real middleware chain.
func TestCreateCategory_AuthMatrix(t *testing.T) {
	body := `{"slug":"honey","name":"Honey"}`

	t.Run("anonymous gets 401", func(t *testing.T) {
		rec := doRequest(newTestServer(newFakeStore()), http.MethodPost, "/api/v1/admin/categories", body, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("customer gets 403", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/admin/categories", body, cookie)
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
	})

	t.Run("admin gets 201", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/admin/categories", body, cookie)
		if rec.Code != http.StatusCreated {
			t.Errorf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
		}
	})
}

func TestCreateCategory_Errors(t *testing.T) {
	fake := newFakeStore()
	fake.categories = []domain.Category{{ID: 1, Slug: "honey", Name: "Honey"}}
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
	server := newTestServer(fake)

	t.Run("validation errors reported per field", func(t *testing.T) {
		rec := doRequest(server, http.MethodPost, "/api/v1/admin/categories", `{"slug":"BAD SLUG","name":""}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		var envelope struct {
			Error struct {
				Code   string            `json:"code"`
				Fields map[string]string `json:"fields"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatalf("bad envelope: %v", err)
		}
		if envelope.Error.Code != "validation_failed" {
			t.Errorf("code = %q, want validation_failed", envelope.Error.Code)
		}
		if _, ok := envelope.Error.Fields["slug"]; !ok {
			t.Errorf("missing slug field error: %v", envelope.Error.Fields)
		}
		if _, ok := envelope.Error.Fields["name"]; !ok {
			t.Errorf("missing name field error: %v", envelope.Error.Fields)
		}
	})

	t.Run("duplicate slug gets 409", func(t *testing.T) {
		rec := doRequest(server, http.MethodPost, "/api/v1/admin/categories", `{"slug":"honey","name":"Honey Again"}`, cookie)
		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
	})

	t.Run("unknown JSON field gets 400", func(t *testing.T) {
		rec := doRequest(server, http.MethodPost, "/api/v1/admin/categories", `{"slug":"x","name":"X","colour":"red"}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestGetProduct_NotFound(t *testing.T) {
	rec := doRequest(newTestServer(newFakeStore()), http.MethodGet, "/api/v1/products/ghost", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
