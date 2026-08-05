package api_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The locale a request resolves to is observable through the store: the fake
// records what the handler passed it, which proves the whole chain —
// middleware → context → handler → store — rather than just the parser.
func localeSeenBy(t *testing.T, req *http.Request) domain.Locale {
	t.Helper()

	fake := newFakeStore()
	srv := newTestServer(fake)

	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("request failed: status %d", rec.Code)
	}
	return fake.lastLocale
}

func TestLocaleNegotiation(t *testing.T) {
	tests := []struct {
		name  string
		build func() *http.Request
		want  domain.Locale
	}{
		{
			name: "defaults to English when nothing is offered",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
			},
			want: domain.LocaleEN,
		},
		{
			name: "query parameter wins",
			build: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/v1/categories?lang=hy", nil)
			},
			want: domain.LocaleHY,
		},
		{
			name: "cookie is used when there is no query parameter",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
				r.AddCookie(&http.Cookie{Name: "mb_locale", Value: "ru"})
				return r
			},
			want: domain.LocaleRU,
		},
		{
			name: "query parameter beats the cookie",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/categories?lang=hy", nil)
				r.AddCookie(&http.Cookie{Name: "mb_locale", Value: "ru"})
				return r
			},
			want: domain.LocaleHY,
		},
		{
			name: "Accept-Language is the last resort",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
				r.Header.Set("Accept-Language", "ru-RU,ru;q=0.9")
				return r
			},
			want: domain.LocaleRU,
		},
		{
			name: "highest q wins, not header order",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
				r.Header.Set("Accept-Language", "en;q=0.3,hy;q=0.9,ru;q=0.5")
				return r
			},
			want: domain.LocaleHY,
		},
		{
			name: "unsupported languages are skipped for a supported one",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
				// German outranks Armenian but we do not speak it.
				r.Header.Set("Accept-Language", "de-DE,de;q=0.9,hy;q=0.4")
				return r
			},
			want: domain.LocaleHY,
		},
		{
			name: "q=0 means explicitly not acceptable",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
				r.Header.Set("Accept-Language", "hy;q=0,ru;q=0.1")
				return r
			},
			want: domain.LocaleRU,
		},
		{
			name: "an unknown value falls through instead of failing",
			build: func() *http.Request {
				// A shop must not refuse to render because a tag was odd.
				return httptest.NewRequest(http.MethodGet, "/api/v1/categories?lang=klingon", nil)
			},
			want: domain.LocaleEN,
		},
		{
			name: "malformed q does not discard the rest of the header",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
				r.Header.Set("Accept-Language", "hy;q=abc,ru;q=0.5")
				return r
			},
			want: domain.LocaleRU,
		},
		{
			name: "wildcard alone means the default",
			build: func() *http.Request {
				r := httptest.NewRequest(http.MethodGet, "/api/v1/categories", nil)
				r.Header.Set("Accept-Language", "*")
				return r
			},
			want: domain.LocaleEN,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := localeSeenBy(t, tc.build()); got != tc.want {
				t.Errorf("resolved locale = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestLocaleReachesProductEndpoints(t *testing.T) {
	// Categories are covered above; these two prove the other handlers wire
	// the locale through as well, rather than defaulting silently.
	t.Run("list", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/api/v1/products?lang=ru", nil)
		if got := localeSeenBy(t, r); got != domain.LocaleRU {
			t.Errorf("list locale = %q; want ru", got)
		}
	})

	t.Run("detail", func(t *testing.T) {
		fake := newFakeStore()
		fake.products = append(fake.products, domain.Product{ID: 1, Slug: "honey", Name: "Honey"})
		srv := newTestServer(fake)

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/products/honey?lang=hy", nil)
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d; want 200", rec.Code)
		}
		if fake.lastLocale != domain.LocaleHY {
			t.Errorf("detail locale = %q; want hy", fake.lastLocale)
		}
	})
}
