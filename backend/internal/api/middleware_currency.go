package api

import (
	"context"
	"net/http"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// currencyCookieName lets a choice made in the footer survive to requests
// the frontend does not control the URL of, and — more importantly — to the
// checkout POST, where the resolved currency decides what the customer is
// billed in.
const currencyCookieName = "mb_currency"

type currencyCtxKey struct{}

// withCurrency resolves the request's market once, at the edge, exactly as
// withLocale resolves its language.
//
// Precedence, most explicit first:
//  1. ?currency=AMD    — an explicit ask, e.g. a shared link
//  2. mb_currency      — the choice the visitor made in the footer
//  3. the resolved locale — an Armenian reader is probably shopping in drams
//  4. USD              — the default
//
// Step 3 is why this middleware must run AFTER withLocale: it consumes the
// language decision rather than re-deriving it from Accept-Language, so
// ?lang=hy and the mb_locale cookie steer the currency guess too.
//
// Nothing here can fail. An unknown code falls through to the next source
// rather than 400ing, which is the same "a shop that refuses to serve a page
// because a header was odd is worse than one that serves it in English"
// argument — with one sharpened edge: the value is never trusted raw. It
// only ever becomes a domain.Currency through ParseCurrency, so what reaches
// the SQL parameter is a value from the whitelist, never the caller's bytes.
func (s *Server) withCurrency(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), currencyCtxKey{}, resolveCurrency(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func resolveCurrency(r *http.Request) domain.Currency {
	if c, ok := domain.ParseCurrency(r.URL.Query().Get("currency")); ok {
		return c
	}
	if cookie, err := r.Cookie(currencyCookieName); err == nil {
		if c, ok := domain.ParseCurrency(cookie.Value); ok {
			return c
		}
	}
	// A GUESS with a get-out: the footer switcher overrides it and the
	// cookie remembers, so being wrong costs one click rather than a sale.
	return domain.CurrencyForLocale(localeFromContext(r.Context()))
}

// currencyFromContext is the handler-facing read; falls back rather than
// reporting absence, so a handler reached without the middleware still
// prices something instead of nothing.
func currencyFromContext(ctx context.Context) domain.Currency {
	if c, ok := ctx.Value(currencyCtxKey{}).(domain.Currency); ok {
		return c
	}
	return domain.DefaultCurrency
}

// viewFromContext bundles the two edge-resolved values into the one thing
// the store actually asks for.
func viewFromContext(ctx context.Context) domain.View {
	return domain.View{
		Locale:   localeFromContext(ctx),
		Currency: currencyFromContext(ctx),
	}
}
