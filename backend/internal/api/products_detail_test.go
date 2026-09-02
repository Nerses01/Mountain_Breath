package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

func adminServer(t *testing.T) (*fakeStore, http.Handler, *http.Cookie) {
	t.Helper()
	fake := newFakeStore()
	fake.products = []domain.Product{{ID: 1, Slug: "honey", Name: "Honey"}}
	srv := newTestServer(fake)
	cookie := loginAs(fake, domain.User{ID: 9, Role: "admin"})
	return fake, srv, cookie
}

// The gallery must nominate exactly one hero. The DATABASE enforces "at most
// one" with a partial unique index; the handler catches "none" and turns
// "two" into a field error a form can attach, rather than a 500-shaped
// constraint violation.
func TestSaveProductImages_RequiresExactlyOnePrimary(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "exactly one",
			body:       `{"images":[{"id":1,"is_primary":true},{"id":2,"is_primary":false}]}`,
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "none nominated",
			body:       `{"images":[{"id":1,"is_primary":false},{"id":2,"is_primary":false}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "two nominated",
			body:       `{"images":[{"id":1,"is_primary":true},{"id":2,"is_primary":true}]}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			// An empty gallery is a legitimate state — a product with no
			// photos yet — so it must not trip the "exactly one" rule.
			name:       "empty gallery",
			body:       `{"images":[]}`,
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, srv, cookie := adminServer(t)
			rec := doRequest(srv, http.MethodPut, "/api/v1/admin/products/1/images", tc.body, cookie)
			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.wantStatus, rec.Body.String())
			}
			if tc.wantStatus == http.StatusBadRequest {
				if !strings.Contains(rec.Body.String(), domain.ValidationOnePrimary) {
					t.Errorf("expected the %s code: %s", domain.ValidationOnePrimary, rec.Body.String())
				}
			}
		})
	}
}

// Position comes from the ARRAY ORDER, not from a sort_order the client also
// sends — the list the admin sees is the list they dragged, and trusting a
// separate number invites the two to disagree.
func TestSaveProductImages_PositionComesFromArrayOrder(t *testing.T) {
	fake, srv, cookie := adminServer(t)

	rec := doRequest(srv, http.MethodPut, "/api/v1/admin/products/1/images", `{"images":[
		{"id":7,"sort_order":99,"is_primary":true,"alt":{"en":"A jar","hy":"Բանկա"}},
		{"id":3,"sort_order":0,"is_primary":false}
	]}`, cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}

	if len(fake.savedImages) != 2 {
		t.Fatalf("saved %d images", len(fake.savedImages))
	}
	if fake.savedImages[0].ID != 7 || fake.savedImages[0].SortOrder != 0 {
		t.Errorf("first image = %+v, want id 7 at position 0", fake.savedImages[0])
	}
	if fake.savedImages[1].SortOrder != 1 {
		t.Errorf("second image position = %d, want 1", fake.savedImages[1].SortOrder)
	}
	// Alt carries "en" as a peer, unlike product translations where English
	// lives in a parent field — an image has no parent field for its alt.
	if got := fake.lastImageAltsByID[7][domain.LocaleEN]; got != "A jar" {
		t.Errorf("English alt = %q", got)
	}
	if got := fake.lastImageAltsByID[7][domain.LocaleHY]; got != "Բանկա" {
		t.Errorf("Armenian alt = %q", got)
	}
}

func TestSaveProductEditorial(t *testing.T) {
	t.Run("keyed by locale, including en", func(t *testing.T) {
		fake, srv, cookie := adminServer(t)

		rec := doRequest(srv, http.MethodPut, "/api/v1/admin/products/1/editorial", `{"content":{
			"en":{"highlights":[{"text":"First"},{"text":"Second"}],
			      "usage_cards":[{"kicker":"Morning","title":"A spoon","body":"Plain."}]},
			"hy":{"highlights":[{"text":"Առաջին"}]}
		}}`, cookie)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}

		en := fake.savedEditorial[domain.LocaleEN]
		if len(en.Highlights) != 2 || en.Highlights[0].Text != "First" {
			t.Errorf("English highlights = %+v", en.Highlights)
		}
		if len(en.UsageCards) != 1 || en.UsageCards[0].Kicker != "Morning" {
			t.Errorf("English usage cards = %+v", en.UsageCards)
		}
		// Languages may legitimately differ in count — decision #4's payoff.
		if len(fake.savedEditorial[domain.LocaleHY].Highlights) != 1 {
			t.Errorf("Armenian highlights = %+v", fake.savedEditorial[domain.LocaleHY].Highlights)
		}
	})

	t.Run("rejects a blank bullet with a field path", func(t *testing.T) {
		_, srv, cookie := adminServer(t)

		rec := doRequest(srv, http.MethodPut, "/api/v1/admin/products/1/editorial",
			`{"content":{"en":{"highlights":[{"text":"Fine"},{"text":"  "}]}}}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}

		var body struct {
			Error struct {
				Fields map[string]string `json:"fields"`
			} `json:"error"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		// The JSON path the form used, so the error lands on the right input.
		if body.Error.Fields["content.en.highlights[1].text"] != domain.ValidationRequired {
			t.Errorf("fields = %v", body.Error.Fields)
		}
	})

	t.Run("rejects an unsupported locale", func(t *testing.T) {
		_, srv, cookie := adminServer(t)

		rec := doRequest(srv, http.MethodPut, "/api/v1/admin/products/1/editorial",
			`{"content":{"de":{"highlights":[{"text":"Erstens"}]}}}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400: %s", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), domain.ValidationLocaleUnsupported) {
			t.Errorf("body = %s", rec.Body.String())
		}
	})
}

func TestSaveProductRelated_KeepsOrder(t *testing.T) {
	fake, srv, cookie := adminServer(t)

	rec := doRequest(srv, http.MethodPut, "/api/v1/admin/products/1/related",
		`{"related_ids":[5,2,9]}`, cookie)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
	}
	// The array's order IS the display order.
	if len(fake.savedRelatedIDs) != 3 || fake.savedRelatedIDs[0] != 5 || fake.savedRelatedIDs[2] != 9 {
		t.Errorf("related ids = %v", fake.savedRelatedIDs)
	}
}

func TestEditorialEndpointsRequireAdmin(t *testing.T) {
	fake := newFakeStore()
	srv := newTestServer(fake)
	customer := loginAs(fake, domain.User{ID: 2, Role: "customer"})

	for _, path := range []string{
		"/api/v1/admin/products/1/images",
		"/api/v1/admin/products/1/editorial",
		"/api/v1/admin/products/1/related",
	} {
		if rec := doRequest(srv, http.MethodPut, path, `{}`, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("anonymous %s = %d, want 401", path, rec.Code)
		}
		if rec := doRequest(srv, http.MethodPut, path, `{}`, customer); rec.Code != http.StatusForbidden {
			t.Errorf("customer %s = %d, want 403", path, rec.Code)
		}
	}
}

// The two catalog endpoints answer different questions, and the payload split
// is the reason both exist. Since decision #99 the split runs through the
// MIDDLE of the media: photos ride on the card (the hover slideshow renders
// them), while the video and the editorial stay detail-only.
func TestDetailCarriesEditorialAndTheListingDoesNot(t *testing.T) {
	video := domain.ProductImage{ID: 9, URL: "/uploads/v.mp4", Alt: "Harvest clip"}
	fake := newFakeStore()
	fake.products = []domain.Product{{
		ID: 1, Slug: "honey", Name: "Honey",
		LabBatch: "WH-0626", IsColdChain: true,
		HarvestNote: "August 2026, Hives 12–18",
		Disclaimer:  "A food, not a medicine.",
		Images:      []domain.ProductImage{{ID: 4, URL: "/uploads/a.jpg", Alt: "A jar", IsPrimary: true}},
		Video:       &video,
		Highlights:  []domain.ProductHighlight{{Text: "Steady natural energy"}},
		UsageCards:  []domain.ProductUsageCard{{Kicker: "Morning", Title: "A spoon", Body: "Plain."}},
	}}
	srv := newTestServer(fake)

	detail := doRequest(srv, http.MethodGet, "/api/v1/products/honey", "", nil).Body.String()
	for _, want := range []string{
		`"lab_batch":"WH-0626"`, `"is_cold_chain":true`,
		`"harvest_note":"August 2026, Hives 12–18"`,
		`"alt":"A jar"`, `"Steady natural energy"`, `"kicker":"Morning"`,
		`"video":{"id":9,"url":"/uploads/v.mp4","alt":"Harvest clip"}`,
	} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail is missing %s", want)
		}
	}

	listing := doRequest(srv, http.MethodGet, "/api/v1/products", "", nil).Body.String()
	for _, unwanted := range []string{"lab_batch", "harvest_note", "usage_cards", "highlights", "video"} {
		if strings.Contains(listing, unwanted) {
			t.Errorf("the card payload carries %q, which no card renders", unwanted)
		}
	}
	// …but the photos DO ride on the card, thin: url and alt only.
	if !strings.Contains(listing, `"images":[{"url":"/uploads/a.jpg","alt":"A jar"}]`) {
		t.Errorf("the card payload is missing its photos: %s", listing)
	}
}

// An unknown slug returns an empty ARRAY, never null: this is a panel on a
// page, and the page itself already 404s if the product is missing.
func TestRelatedProductsAlwaysReturnsAnArray(t *testing.T) {
	fake := newFakeStore()
	srv := newTestServer(fake)

	rec := doRequest(srv, http.MethodGet, "/api/v1/products/nope/related", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("body = %s, want []", got)
	}
}

// ?curated=true is what the admin's picker reads. The distinction is not
// cosmetic: pre-filling a picker from the normal read would present the
// COMPUTED list as curated, and saving it would freeze a dynamic panel — so
// the two must not be the same request.
func TestRelatedProducts_CuratedOnly(t *testing.T) {
	fake := newFakeStore()
	fake.related = []domain.Product{{ID: 2, Slug: "computed"}}
	fake.curatedRelated = []domain.Product{{ID: 3, Slug: "curated"}}
	srv := newTestServer(fake)

	rec := doRequest(srv, http.MethodGet, "/api/v1/products/honey/related", "", nil)
	if !strings.Contains(rec.Body.String(), "computed") {
		t.Errorf("default read = %s, want the computed fallback", rec.Body.String())
	}
	if fake.curatedRelatedAsked {
		t.Error("the storefront read asked for the curated-only list")
	}

	rec = doRequest(srv, http.MethodGet, "/api/v1/products/honey/related?curated=true", "", nil)
	if !strings.Contains(rec.Body.String(), "curated") ||
		strings.Contains(rec.Body.String(), "computed") {
		t.Errorf("curated read = %s", rec.Body.String())
	}
}
