package api

import (
	"encoding/xml"
	"net/http"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// cacheControl wraps a handler with one header — middleware in its smallest
// possible form, for the two places (uploads, sitemap) that want it.
func cacheControl(value string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", value)
		next.ServeHTTP(w, r)
	})
}

// The static storefront routes worth indexing, locale-less; the handler
// crosses them with every locale. Deliberately absent: cart, checkout,
// account, wishlist (private or session-shaped — a crawler in a checkout
// is noise), and the individual JOURNAL posts — their slugs live in
// frontend markdown the backend cannot see, a known cost of decision #77
// recorded there; the /journal list page carries their links for crawlers
// that render.
var sitemapStaticPaths = []string{
	"/", "/shop", "/our-hive", "/benefits", "/shipping", "/contact",
	"/terms", "/privacy", "/journal", "/login",
}

type sitemapURL struct {
	Loc string `xml:"loc"`
}

type sitemapSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []sitemapURL `xml:"url"`
}

// GET /sitemap.xml — every public page in every language. One <url> per
// (path, locale); the per-page hreflang cross-references already live in
// each page's <head> (usePageMeta), which is the arrangement Google
// documents as sufficient.
func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	slugs, err := s.store.ListProductSlugs(r.Context())
	if err != nil {
		s.log.Error("listing slugs for sitemap", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	paths := make([]string, 0, len(sitemapStaticPaths)+len(slugs))
	paths = append(paths, sitemapStaticPaths...)
	for _, slug := range slugs {
		paths = append(paths, "/products/"+slug)
	}

	set := sitemapSet{XMLNS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	for _, locale := range domain.Locales {
		prefix := localePathPrefix(locale)
		for _, p := range paths {
			loc := s.publicURL + prefix + p
			if p == "/" {
				// The bare locale roots: "/", "/hy", "/ru" — not "/hy/".
				loc = s.publicURL + prefix
				if prefix == "" {
					loc = s.publicURL + "/"
				}
			}
			set.URLs = append(set.URLs, sitemapURL{Loc: loc})
		}
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write([]byte(xml.Header))
	if err := xml.NewEncoder(w).Encode(set); err != nil {
		s.log.Error("encoding sitemap", "error", err)
	}
}
