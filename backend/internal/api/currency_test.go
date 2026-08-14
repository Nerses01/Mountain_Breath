package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// Same trick as localeSeenBy: the resolved currency is observable through the
// store, which proves the whole chain — middleware → context → handler →
// query — rather than just the parser.
func currencySeenBy(t *testing.T, req *http.Request) domain.Currency {
	t.Helper()

	fake := newFakeStore()
	fake.products = []domain.Product{{ID: 1, Slug: "honey", Name: "Honey"}}
	srv := newTestServer(fake)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("request failed: status %d (%s)", rec.Code, rec.Body.String())
	}
	return fake.lastCurrency
}

func TestCurrencyNegotiation(t *testing.T) {
	const path = "/api/v1/products/honey"

	tests := []struct {
		name  string
		build func() *http.Request
		want  domain.Currency
	}{
		{
			name: "defaults to the base currency when nothing is offered",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, path, nil)
			},
			want: domain.CurrencyUSD,
		},
		{
			name: "query parameter wins",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, path+"?currency=AMD", nil)
			},
			want: domain.CurrencyAMD,
		},
		{
			name: "lowercase is a person typing, not an attack",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, path+"?currency=amd", nil)
			},
			want: domain.CurrencyAMD,
		},
		{
			name: "cookie is used when there is no query parameter",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, path, nil)
				r.AddCookie(&http.Cookie{Name: "mb_currency", Value: "AMD"})
				return r
			},
			want: domain.CurrencyAMD,
		},
		{
			name: "query parameter beats the cookie",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, path+"?currency=USD", nil)
				r.AddCookie(&http.Cookie{Name: "mb_currency", Value: "AMD"})
				return r
			},
			want: domain.CurrencyUSD,
		},
		{
			// The step that makes middleware ORDER matter: the currency
			// guess consumes the language decision, so ?lang=hy steers it
			// even though it says nothing about money.
			name: "an Armenian reader is guessed to be an Armenian shopper",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, path+"?lang=hy", nil)
			},
			want: domain.CurrencyAMD,
		},
		{
			name: "Accept-Language reaches the guess too",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, path, nil)
				r.Header.Set("Accept-Language", "hy-AM,hy;q=0.9,en;q=0.5")
				return r
			},
			want: domain.CurrencyAMD,
		},
		{
			// Russian is deliberately not mapped: a Russian speaker may be
			// reading from anywhere.
			name: "a language with no market guess falls back to the default",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, path+"?lang=ru", nil)
			},
			want: domain.CurrencyUSD,
		},
		{
			name: "an unknown code falls back instead of 400ing",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, path+"?currency=EUR", nil)
			},
			want: domain.CurrencyUSD,
		},
		{
			// The value never reaches SQL as text — ParseCurrency is the only
			// door into a domain.Currency, and it answers with a whitelisted
			// value or the default.
			name: "an injection attempt is simply not a currency",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet,
					path+"?currency=USD%27%3B%20DROP%20TABLE%20variant_prices%3B%20--", nil)
			},
			want: domain.CurrencyUSD,
		},
		{
			name: "a garbage cookie falls through to the language guess",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, path+"?lang=hy", nil)
				r.AddCookie(&http.Cookie{Name: "mb_currency", Value: "not-a-currency"})
				return r
			},
			want: domain.CurrencyAMD,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := currencySeenBy(t, tc.build()); got != tc.want {
				t.Errorf("resolved currency = %q, want %q", got, tc.want)
			}
		})
	}
}

// The catalog's price bounds are denominated in the resolved currency, so the
// two have to be parsed out of the same query string together. Reading
// max_price without the currency would mean "$20" and "20 ֏" were the same
// filter.
func TestPriceFilterTravelsWithItsCurrency(t *testing.T) {
	fake := newFakeStore()
	srv := newTestServer(fake)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/v1/products?currency=AMD&min_price=4000&max_price=9000", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	f := fake.lastFilter
	if f.EffectiveCurrency() != domain.CurrencyAMD {
		t.Errorf("filter currency = %q, want AMD", f.EffectiveCurrency())
	}
	if f.PriceMinMinor == nil || *f.PriceMinMinor != 4000 ||
		f.PriceMaxMinor == nil || *f.PriceMaxMinor != 9000 {
		t.Errorf("bounds did not reach the filter: %+v", f)
	}
}

// The design shows a primary price and a muted second one, so the JSON has to
// carry both — and say which is which.
func TestProductResponseCarriesBothMarkets(t *testing.T) {
	fake := newFakeStore()
	fake.products = []domain.Product{{
		ID: 1, Slug: "honey", Name: "Honey",
		Variants: []domain.ProductVariant{{
			ID: 1, SKU: "HON-1", Label: "500 g", StockQty: 4,
			PriceMinor: 6700,
			Prices:     domain.Money{domain.CurrencyUSD: 1400, domain.CurrencyAMD: 6700},
		}},
	}}

	rec := httptest.NewRecorder()
	newTestServer(fake).ServeHTTP(rec,
		httptest.NewRequest(http.MethodGet, "/api/v1/products/honey?currency=AMD", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}

	var got struct {
		Currency string `json:"currency"`
		Variants []struct {
			PriceMinor int64            `json:"price_minor"`
			Prices     map[string]int64 `json:"prices"`
		} `json:"variants"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	if got.Currency != "AMD" {
		t.Errorf("currency = %q, want AMD", got.Currency)
	}
	if got.Variants[0].PriceMinor != 6700 {
		t.Errorf("price_minor = %d, want the AMD price 6700", got.Variants[0].PriceMinor)
	}
	if got.Variants[0].Prices["USD"] != 1400 || got.Variants[0].Prices["AMD"] != 6700 {
		t.Errorf("prices = %v, want both markets", got.Variants[0].Prices)
	}
}

// An order is charged in ONE currency, and the client does not get to name
// it. A body that tried would be rejected as unknown JSON; this proves the
// edge-resolved value is what the store is told. (The body itself is E6's
// checkout shape — cash on delivery, which is legal precisely because the
// resolved currency IS drams.)
func TestCheckoutChargesTheResolvedCurrency(t *testing.T) {
	fake := newFakeStore()
	fake.cart = []domain.CartItem{{
		VariantID: 1, Qty: 1, PriceMinor: 6700,
		Prices: domain.Money{domain.CurrencyUSD: 1400, domain.CurrencyAMD: 6700},
	}}
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

	body := `{
		"address": {"first_name": "Anahit", "last_name": "Sargsyan",
		            "phone": "+374 91 000000", "street": "14 Abovyan St",
		            "city": "Yerevan", "postal_code": "0009", "country": "AM"},
		"payment_method": "cash_on_delivery"
	}`
	rec := doRequest(newTestServer(fake), http.MethodPost, "/api/v1/orders?currency=AMD", body, cookie)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}
	if fake.lastCurrency != domain.CurrencyAMD {
		t.Errorf("store was told %q, want AMD", fake.lastCurrency)
	}

	var got struct {
		Currency   string  `json:"currency"`
		FxRateUsed *string `json:"fx_rate_used"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Currency != "AMD" {
		t.Errorf("order currency = %q, want AMD", got.Currency)
	}
	// omitempty: a base-currency order (this fake returns no rate) says
	// "not applicable" by leaving the key out, not by sending 1.0.
	if got.FxRateUsed != nil {
		t.Errorf("fx_rate_used = %q, want the key omitted", *got.FxRateUsed)
	}
}

// The cart totals every market separately, and the response says which one
// the flat total_minor belongs to. Since E6 the totals INCLUDE shipping —
// quoted per market from shipping_rates, never converted — and the
// subtotals keep the old sum-of-lines meaning.
func TestCartResponseTotalsEachMarket(t *testing.T) {
	fake := newFakeStore()
	fake.cart = []domain.CartItem{
		{VariantID: 1, Qty: 2, PriceMinor: 6700,
			Prices: domain.Money{domain.CurrencyUSD: 1400, domain.CurrencyAMD: 6700}},
		{VariantID: 2, Qty: 1, PriceMinor: 15300,
			Prices: domain.Money{domain.CurrencyUSD: 3200, domain.CurrencyAMD: 15300}},
	}
	cookie := loginAs(fake, domain.User{ID: 1, Role: domain.RoleCustomer})

	rec := doRequest(newTestServer(fake), http.MethodGet, "/api/v1/cart?currency=AMD", "", cookie)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", rec.Code, rec.Body.String())
	}

	var got struct {
		Currency      string           `json:"currency"`
		SubtotalMinor int64            `json:"subtotal_minor"`
		ShippingMinor int64            `json:"shipping_minor"`
		TotalMinor    int64            `json:"total_minor"`
		Subtotals     map[string]int64 `json:"subtotals"`
		Totals        map[string]int64 `json:"totals"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}

	// 28,700 ֏ of lines + the 1,900 ֏ base rate (under the 33,500 free
	// threshold, nothing cold-chain in this basket).
	if got.Currency != "AMD" || got.SubtotalMinor != 28700 ||
		got.ShippingMinor != 1900 || got.TotalMinor != 30600 {
		t.Errorf("resolved quote = %d + %d = %d %s, want 28700 + 1900 = 30600 AMD",
			got.SubtotalMinor, got.ShippingMinor, got.TotalMinor, got.Currency)
	}
	// The dollar column is quoted with the DOLLAR rate ($4 base), not the
	// dram fee converted.
	if got.Subtotals["USD"] != 6000 || got.Totals["USD"] != 6400 {
		t.Errorf("USD quote = %d → %d, want 6000 → 6400", got.Subtotals["USD"], got.Totals["USD"])
	}
}
