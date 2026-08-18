package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// A2's handler contract: reorder is a POST that answers 200 with a line-by-
// line report — partial success included — and hides strangers' orders
// behind the same 404 as the order page. The MERGE logic itself (caps,
// skips) lives in the store and is covered by the Docker-backed suite.
func TestReorderHandler(t *testing.T) {
	fake := newFakeStore()
	fake.orders = []domain.Order{{ID: 12, UserID: 1, Status: domain.OrderDelivered}}
	fake.reorderResult = domain.ReorderResult{Lines: []domain.ReorderLine{
		{Name: "Wild Honey", Label: "700 g", Qty: 2},
		{Name: "Fresh Comb", Label: "340 g", Qty: 0, Issue: domain.ReorderOutOfStock},
	}}

	t.Run("the owner gets the merge report", func(t *testing.T) {
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders/12/reorder", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}

		var got struct {
			Lines []struct {
				Name  string `json:"name"`
				Qty   int    `json:"qty"`
				Issue string `json:"issue"`
			} `json:"lines"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if len(got.Lines) != 2 {
			t.Fatalf("lines = %d, want 2 (%s)", len(got.Lines), rec.Body.String())
		}
		if got.Lines[0].Qty != 2 || got.Lines[0].Issue != "" {
			t.Errorf("added line wrong: %+v", got.Lines[0])
		}
		if got.Lines[1].Qty != 0 || got.Lines[1].Issue != "out_of_stock" {
			t.Errorf("skipped line wrong: %+v", got.Lines[1])
		}
		if fake.reorderUser != 1 || fake.reorderOrder != 12 {
			t.Errorf("store asked for user %d order %d", fake.reorderUser, fake.reorderOrder)
		}
	})

	t.Run("a stranger's order id is a 404, not a 403", func(t *testing.T) {
		cookie := loginAs(fake, domain.User{ID: 2, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders/12/reorder", "", cookie)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want 404 (%s)", rec.Code, rec.Body.String())
		}
	})

	t.Run("anonymous is a 401 from requireUser", func(t *testing.T) {
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders/12/reorder", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}
