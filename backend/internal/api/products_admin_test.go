package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

const validProductBody = `{
	"category_id": 1,
	"slug": "mountain-jam",
	"name": "Mountain Jam",
	"description": "Wild berry jam.",
	"variants": [
		{"sku": "JAM-1", "label": "300 g", "prices": {"USD": 2500, "AMD": 9800}, "stock_qty": 10}
	]
}`

func TestCreateProduct(t *testing.T) {
	t.Run("anonymous gets 401", func(t *testing.T) {
		rec := doRequest(newTestServer(newFakeStore()), http.MethodPost, "/api/v1/admin/products", validProductBody, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("admin creates product", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/admin/products", validProductBody, cookie)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201 (body: %s)", rec.Code, rec.Body.String())
		}
		var got struct {
			IsActive bool `json:"is_active"`
			Variants []struct {
				SKU string `json:"sku"`
			} `json:"variants"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if !got.IsActive || len(got.Variants) != 1 {
			t.Errorf("unexpected response: %s", rec.Body.String())
		}
	})

	t.Run("validation errors use variants[i] field paths", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
		body := `{"category_id": 1, "slug": "x y", "name": "", "variants": [{"sku": "", "label": "L", "prices": {"USD": 0, "XXX": 5}, "stock_qty": -1}]}`
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/admin/products", body, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		var envelope struct {
			Error struct {
				Fields map[string]string `json:"fields"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"name", "slug", "variants[0].sku", "variants[0].prices.USD", "variants[0].prices.XXX", "variants[0].stock_qty"} {
			if _, ok := envelope.Error.Fields[key]; !ok {
				t.Errorf("missing field error %q in %v", key, envelope.Error.Fields)
			}
		}
	})

	t.Run("duplicate sku gets 409", func(t *testing.T) {
		fake := newFakeStore()
		fake.products = []domain.Product{{
			ID: 1, Slug: "existing", Variants: []domain.ProductVariant{{SKU: "JAM-1"}},
		}}
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/admin/products", validProductBody, cookie)
		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409 (body: %s)", rec.Code, rec.Body.String())
		}
	})
}

func TestUpdateVariant(t *testing.T) {
	fake := newFakeStore()
	fake.products = []domain.Product{{
		ID: 1, Slug: "p", Variants: []domain.ProductVariant{{ID: 7, SKU: "S", PriceMinor: 100, StockQty: 1}},
	}}
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleAdmin})
	server := newTestServer(fake)

	rec := doRequest(server, http.MethodPatch, "/api/v1/admin/variants/7", `{"prices": {"USD": 900, "AMD": 3500}, "stock_qty": 5}`, cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 (body: %s)", rec.Code, rec.Body.String())
	}
	got := fake.products[0].Variants[0]
	if got.Prices[domain.CurrencyUSD] != 900 || got.Prices[domain.CurrencyAMD] != 3500 || got.StockQty != 5 {
		t.Errorf("variant not updated: %+v", got)
	}

	rec = doRequest(server, http.MethodPatch, "/api/v1/admin/variants/999", `{"prices": {"USD": 900}, "stock_qty": 5}`, cookie)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}

	// The base currency is not optional: without it nothing can be priced,
	// not even by conversion, so a prices map that omits it is a 400 rather
	// than a variant that quietly disappears from the shop.
	rec = doRequest(server, http.MethodPatch, "/api/v1/admin/variants/7", `{"prices": {"AMD": 3500}, "stock_qty": 5}`, cookie)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing base price: status = %d, want 400 (body: %s)", rec.Code, rec.Body.String())
	}
}
