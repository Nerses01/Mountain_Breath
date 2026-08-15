package api_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

func TestSitemap(t *testing.T) {
	fake := newFakeStore()
	fake.products = []domain.Product{{ID: 1, Slug: "mountain-wildflower-honey"}}
	srv := newTestServerOpts(fake, api.Options{PublicURL: "https://shop.example"})

	rec := doRequest(srv, http.MethodGet, "/sitemap.xml", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/xml") {
		t.Errorf("content type = %q", ct)
	}

	body := rec.Body.String()
	// Every language's twin of a product page, on the PUBLIC origin.
	for _, url := range []string{
		"<loc>https://shop.example/products/mountain-wildflower-honey</loc>",
		"<loc>https://shop.example/hy/products/mountain-wildflower-honey</loc>",
		"<loc>https://shop.example/ru/products/mountain-wildflower-honey</loc>",
		"<loc>https://shop.example/shop</loc>",
		"<loc>https://shop.example/hy</loc>",
	} {
		if !strings.Contains(body, url) {
			t.Errorf("sitemap missing %s", url)
		}
	}
	// The private routes stay out — a crawler in a checkout is noise.
	for _, absent := range []string{"/cart", "/checkout", "/account", "/admin"} {
		if strings.Contains(body, "shop.example"+absent) {
			t.Errorf("sitemap leaked the private route %s", absent)
		}
	}
}
