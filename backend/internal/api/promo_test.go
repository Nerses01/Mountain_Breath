package api_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The wire shape of a preview, as the tests read it back.
type previewBody struct {
	Currency            string           `json:"currency"`
	SubtotalMinor       int64            `json:"subtotal_minor"`
	ShippingMinor       int64            `json:"shipping_minor"`
	MemberDiscountMinor int64            `json:"member_discount_minor"`
	PromoDiscountMinor  int64            `json:"promo_discount_minor"`
	DiscountMinor       int64            `json:"discount_minor"`
	TaxMinor            int64            `json:"tax_minor"`
	TotalMinor          int64            `json:"total_minor"`
	FirstDeliveryFree   bool             `json:"first_delivery_free"`
	BaseShippingWaived  bool             `json:"base_shipping_waived"`
	RemainingMinor      *int64           `json:"free_shipping_remaining_minor"`
	ThresholdMinor      *int64           `json:"free_shipping_threshold_minor"`
	PromoCode           string           `json:"promo_code"`
	PromoIssue          string           `json:"promo_issue"`
	Totals              map[string]int64 `json:"totals"`
	Upsell              *struct {
		Slug       string `json:"slug"`
		PriceMinor int64  `json:"price_minor"`
	} `json:"upsell"`
}

func decodePreview(t *testing.T, body []byte) previewBody {
	t.Helper()
	var p previewBody
	if err := json.Unmarshal(body, &p); err != nil {
		t.Fatalf("decoding preview: %v (%s)", err, body)
	}
	return p
}

func fieldErrors(t *testing.T, body []byte) map[string]string {
	t.Helper()
	var envelope struct {
		Error struct {
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Error.Fields
}

func TestCheckoutPreview(t *testing.T) {
	t.Run("first order: base waived, no bar, no discounts", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem() // $14.00 / 6,700 ֏
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/checkout/preview", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		p := decodePreview(t, rec.Body.Bytes())

		if p.SubtotalMinor != 1400 || p.ShippingMinor != 0 || p.TotalMinor != 1400 {
			t.Errorf("figures = %d/%d/%d, want 1400/0/1400",
				p.SubtotalMinor, p.ShippingMinor, p.TotalMinor)
		}
		if !p.FirstDeliveryFree || !p.BaseShippingWaived {
			t.Error("first delivery not marked free")
		}
		// The bar counts toward a base that is not being charged — absent.
		if p.RemainingMinor != nil || p.Upsell != nil {
			t.Error("progress bar sent despite waived base")
		}
		// Both markets priced, independently: the dual line's data.
		if p.Totals["USD"] != 1400 || p.Totals["AMD"] != 6700 {
			t.Errorf("totals = %v", p.Totals)
		}
	})

	t.Run("member: 8%, the bar, and the upsell", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem()
		fake.priorOrders = 2
		fake.upsell = &domain.Upsell{Slug: "bee-pollen-granules", Name: "Bee Pollen Granules", PriceMinor: 1600}
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/checkout/preview", "", cookie)
		p := decodePreview(t, rec.Body.Bytes())

		if p.MemberDiscountMinor != 112 { // 8% of $14.00
			t.Errorf("member discount = %d, want 112", p.MemberDiscountMinor)
		}
		if p.ShippingMinor != 400 || p.TotalMinor != 1688 {
			t.Errorf("shipping/total = %d/%d, want 400/1688", p.ShippingMinor, p.TotalMinor)
		}
		if p.RemainingMinor == nil || *p.RemainingMinor != 5600 ||
			p.ThresholdMinor == nil || *p.ThresholdMinor != 7000 {
			t.Errorf("bar = %v/%v, want 5600/7000", p.RemainingMinor, p.ThresholdMinor)
		}
		if p.Upsell == nil || p.Upsell.Slug != "bee-pollen-granules" {
			t.Errorf("upsell = %+v", p.Upsell)
		}
	})

	t.Run("empty cart previews as zeros, not an error", func(t *testing.T) {
		fake := newFakeStore()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/checkout/preview", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if p := decodePreview(t, rec.Body.Bytes()); p.TotalMinor != 0 || p.ShippingMinor != 0 {
			t.Errorf("empty cart priced: %+v", p)
		}
	})

	t.Run("anonymous gets 401", func(t *testing.T) {
		rec := doRequest(newTestServer(newFakeStore()), http.MethodPost, "/api/v1/checkout/preview", "", nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})
}

func TestApplyPromo(t *testing.T) {
	honey10 := domain.Promo{ID: 7, Code: "HONEY10", Kind: domain.PromoPercent, Percent: 10, Active: true}

	t.Run("a valid code applies and the fresh preview carries it", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem()
		fake.priorOrders = 1
		fake.promos = map[string]domain.Promo{"HONEY10": honey10}
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		// Lowercase with whitespace on purpose: normalization is part of the
		// contract, not a client courtesy.
		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/cart/promo",
			`{"code": "  honey10 "}`, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		p := decodePreview(t, rec.Body.Bytes())
		if p.PromoCode != "HONEY10" || p.PromoDiscountMinor != 140 {
			t.Errorf("promo = %q/%d, want HONEY10/140", p.PromoCode, p.PromoDiscountMinor)
		}
		// Member 8% (112) and the code (140) stack side by side.
		if p.DiscountMinor != 252 || p.TotalMinor != 1400+400-252 {
			t.Errorf("discount/total = %d/%d", p.DiscountMinor, p.TotalMinor)
		}
		if fake.cartPromo == nil || fake.cartPromo.ID != 7 {
			t.Error("promo not attached to the cart")
		}
	})

	t.Run("an unknown code is a field error under the box", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem()
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/cart/promo",
			`{"code": "NOPE"}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if fields := fieldErrors(t, rec.Body.Bytes()); fields["promo_code"] != "promo_unknown" {
			t.Errorf("fields = %v, want promo_code: promo_unknown", fields)
		}
	})

	t.Run("a basket under the code's floor is refused at apply time", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem() // $14.00
		floor := int64(3000)
		code := honey10
		code.Values = map[domain.Currency]domain.PromoValue{
			domain.CurrencyUSD: {MinSubtotalMinor: &floor}, // over $30 only
		}
		fake.promos = map[string]domain.Promo{"HONEY10": code}
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/cart/promo",
			`{"code": "HONEY10"}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if fields := fieldErrors(t, rec.Body.Bytes()); fields["promo_code"] != "promo_min_subtotal" {
			t.Errorf("fields = %v, want promo_code: promo_min_subtotal", fields)
		}
		if fake.cartPromo != nil {
			t.Error("an inapplicable code was attached anyway")
		}
	})

	t.Run("removing the code answers with a promo-less preview", func(t *testing.T) {
		fake := newFakeStore()
		fake.cart = cartWithOneItem()
		fake.promos = map[string]domain.Promo{"HONEY10": honey10}
		fake.cartPromo = &honey10
		cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

		rec := doRequest(newTestServer(fake), http.MethodDelete, "/api/v1/cart/promo", "", cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
		}
		if p := decodePreview(t, rec.Body.Bytes()); p.PromoCode != "" || p.PromoDiscountMinor != 0 {
			t.Errorf("promo survived removal: %+v", p)
		}
		if fake.cartPromo != nil {
			t.Error("cart promo not cleared")
		}
	})
}

// The checkout-time refusal: a code that died between apply and "Place the
// order" is a 409, never a silent repricing — the customer approved a total
// this order would no longer match.
func TestCheckoutRefusesStalePromo(t *testing.T) {
	fake := newFakeStore()
	fake.cart = cartWithOneItem()
	fake.orderErr = fmt.Errorf("%w: HONEY10 (%s)",
		domain.ErrPromoInvalid, domain.ValidationPromoExhausted)
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

	rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders", validCheckoutBody, cookie)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409 (%s)", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Code != "promo_invalid" {
		t.Errorf("code = %q, want promo_invalid", envelope.Error.Code)
	}
}

// The header badge's data source: the hive standing rides on /auth/me, and
// the SERVER derives it — the client renders booleans, it does not
// re-implement "after the first order".
func TestMeCarriesHiveStanding(t *testing.T) {
	fake := newFakeStore()
	fake.priorOrders = 3
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

	rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/auth/me", "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var got struct {
		Hive struct {
			PriorOrders           int  `json:"prior_orders"`
			Member                bool `json:"member"`
			MemberDiscountPercent int  `json:"member_discount_percent"`
			FirstDeliveryFree     bool `json:"first_delivery_free"`
		} `json:"hive"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Hive.Member || got.Hive.MemberDiscountPercent != 8 ||
		got.Hive.FirstDeliveryFree || got.Hive.PriorOrders != 3 {
		t.Errorf("hive = %+v", got.Hive)
	}
}
