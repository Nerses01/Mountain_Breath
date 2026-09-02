package api

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// localeCookieName lets a choice made in the UI survive to requests the
// frontend does not control the URL of (image links, a shared API call).
const localeCookieName = "mb_locale"

type localeCtxKey struct{}

// withLocale resolves the request's language once, at the edge, and puts it
// in the context so no handler has to re-derive it.
//
// Precedence, most explicit first:
//  1. ?lang=hy      — an explicit ask, e.g. a shared link
//  2. mb_locale     — a choice the visitor made earlier
//  3. Accept-Language — what the browser says they read
//  4. English       — the default
//
// Nothing here can fail: an unknown or malformed value falls through to the
// next source rather than 400ing. A shop that refuses to serve a page
// because a header was odd is worse than one that serves it in English.
func (s *Server) withLocale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		locale := resolveLocale(r)
		ctx := context.WithValue(r.Context(), localeCtxKey{}, locale)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func resolveLocale(r *http.Request) domain.Locale {
	if l, ok := domain.ParseLocale(r.URL.Query().Get("lang")); ok {
		return l
	}
	if c, err := r.Cookie(localeCookieName); err == nil {
		if l, ok := domain.ParseLocale(c.Value); ok {
			return l
		}
	}
	if l, ok := localeFromAcceptLanguage(r.Header.Get("Accept-Language")); ok {
		return l
	}
	return domain.DefaultLocale
}

// localeFromContext is the handler-facing read. Falls back rather than
// reporting absence: a handler reached without the middleware (a test, a
// future route) should still render something.
func localeFromContext(ctx context.Context) domain.Locale {
	if l, ok := ctx.Value(localeCtxKey{}).(domain.Locale); ok {
		return l
	}
	return domain.DefaultLocale
}

// localeFromAcceptLanguage picks the best supported language out of a header
// like "hy-AM,hy;q=0.9,ru;q=0.8,en;q=0.7".
//
// Hand-rolled rather than pulling in golang.org/x/text/language: the full
// spec covers script subtags and wildcard matching this shop has no use for,
// and the rule here is small enough to read in one screen — quality values,
// highest first, first supported wins.
func localeFromAcceptLanguage(header string) (domain.Locale, bool) {
	if header == "" {
		return domain.DefaultLocale, false
	}

	type candidate struct {
		locale domain.Locale
		q      float64
		order  int // preserves header order for equal q, as the spec expects
	}
	var candidates []candidate

	for i, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		tag, q := part, 1.0
		if name, params, found := strings.Cut(part, ";"); found {
			tag = strings.TrimSpace(name)
			// Only q is meaningful to us; any other parameter is ignored.
			if _, qv, ok := strings.Cut(strings.TrimSpace(params), "q="); ok {
				parsed, err := strconv.ParseFloat(strings.TrimSpace(qv), 64)
				if err != nil {
					continue // malformed q — drop this candidate, keep the rest
				}
				q = parsed
			}
		}

		// "*" means "anything", which tells us nothing we do not already do
		// by defaulting.
		if tag == "*" {
			continue
		}
		// q=0 explicitly means "not acceptable".
		if q <= 0 {
			continue
		}
		if l, ok := domain.ParseLocale(tag); ok {
			candidates = append(candidates, candidate{locale: l, q: q, order: i})
		}
	}

	if len(candidates) == 0 {
		return domain.DefaultLocale, false
	}
	sort.SliceStable(candidates, func(a, b int) bool {
		if candidates[a].q != candidates[b].q {
			return candidates[a].q > candidates[b].q
		}
		return candidates[a].order < candidates[b].order
	})
	return candidates[0].locale, true
}
