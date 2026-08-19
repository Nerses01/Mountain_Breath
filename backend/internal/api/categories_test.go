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

// F2 (decision #95): the category management endpoints. The rules live in
// domain/store; what the handlers owe is the admin gate, the error
// vocabulary (409 slug_taken / 409 category_in_use / 404), and the
// editor's list shape (raw English + translations).
func TestCategoryAdmin(t *testing.T) {
	seeded := func() *fakeStore {
		fake := newFakeStore()
		fake.categories = []domain.Category{
			{ID: 1, Slug: "honey", Name: "Honey", SortOrder: 10,
				Translations: map[domain.Locale]string{domain.LocaleHY: "Մեղր"}},
			{ID: 2, Slug: "tea", Name: "Tea", SortOrder: 20},
		}
		return fake
	}
	admin := func(fake *fakeStore) *http.Cookie {
		return loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
	}

	t.Run("every route is admin-gated", func(t *testing.T) {
		fake := seeded()
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleCustomer})
		for _, req := range [][2]string{
			{http.MethodGet, "/api/v1/admin/categories"},
			{http.MethodPut, "/api/v1/admin/categories/order"},
			{http.MethodPut, "/api/v1/admin/categories/1"},
			{http.MethodDelete, "/api/v1/admin/categories/1"},
		} {
			if rec := doRequest(newTestServer(fake), req[0], req[1], "", nil); rec.Code != http.StatusUnauthorized {
				t.Errorf("%s %s anonymous: %d, want 401", req[0], req[1], rec.Code)
			}
			if rec := doRequest(newTestServer(fake), req[0], req[1], "", cookie); rec.Code != http.StatusForbidden {
				t.Errorf("%s %s customer: %d, want 403", req[0], req[1], rec.Code)
			}
		}
	})

	t.Run("the editor's list carries raw English and translations", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/admin/categories", "", admin(fake))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d", rec.Code)
		}
		var got []struct {
			Name         string            `json:"name"`
			Translations map[string]string `json:"translations"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got) != 2 || got[0].Translations["hy"] != "Մեղր" {
			t.Errorf("list = %+v, want the Armenian translation on honey", got)
		}
	})

	t.Run("update round-trips and refuses a stolen slug", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodPut, "/api/v1/admin/categories/2",
			`{"slug":"herbal-tea","name":"Herbal Tea","sort_order":20}`, admin(fake))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if fake.categories[1].Slug != "herbal-tea" {
			t.Errorf("stored slug = %q", fake.categories[1].Slug)
		}
		rec = doRequest(newTestServer(fake), http.MethodPut, "/api/v1/admin/categories/2",
			`{"slug":"honey","name":"Tea","sort_order":20}`, admin(fake))
		if rec.Code != http.StatusConflict {
			t.Errorf("stolen slug: status = %d, want 409; body %s", rec.Code, rec.Body)
		}
	})

	t.Run("an en translations key is rejected on update too", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodPut, "/api/v1/admin/categories/1",
			`{"slug":"honey","name":"Honey","sort_order":10,"translations":{"en":"Honey"}}`, admin(fake))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400 (en lives in name)", rec.Code)
		}
	})

	t.Run("delete: 204 empty, 409 with products, 404 ghost", func(t *testing.T) {
		fake := seeded()
		fake.products = []domain.Product{{ID: 5, CategoryID: 1, Slug: "wild-honey", Name: "Wild Honey"}}
		if rec := doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/admin/categories/1", "", admin(fake)); rec.Code != http.StatusConflict {
			t.Errorf("in use: status = %d, want 409; body %s", rec.Code, rec.Body)
		}
		if rec := doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/admin/categories/2", "", admin(fake)); rec.Code != http.StatusNoContent {
			t.Errorf("empty: status = %d, want 204", rec.Code)
		}
		if rec := doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/admin/categories/99", "", admin(fake)); rec.Code != http.StatusNotFound {
			t.Errorf("ghost: status = %d, want 404", rec.Code)
		}
	})

	t.Run("reorder applies by position; a stale id is 404; empty list 400", func(t *testing.T) {
		fake := seeded()
		rec := doRequest(newTestServer(fake), http.MethodPut, "/api/v1/admin/categories/order",
			`{"ids":[2,1]}`, admin(fake))
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d; body %s", rec.Code, rec.Body)
		}
		if fake.categories[0].SortOrder != 20 || fake.categories[1].SortOrder != 10 {
			t.Errorf("sort orders = %d, %d — tea should now come first",
				fake.categories[0].SortOrder, fake.categories[1].SortOrder)
		}
		rec = doRequest(newTestServer(fake), http.MethodPut, "/api/v1/admin/categories/order",
			`{"ids":[1,99]}`, admin(fake))
		if rec.Code != http.StatusNotFound {
			t.Errorf("stale id: status = %d, want 404", rec.Code)
		}
		rec = doRequest(newTestServer(fake), http.MethodPut, "/api/v1/admin/categories/order",
			`{"ids":[]}`, admin(fake))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("empty list: status = %d, want 400", rec.Code)
		}
	})
}
