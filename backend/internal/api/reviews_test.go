package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The public list must never be able to ask for unmoderated rows, however
// the URL is edited.
func TestListReviews_PublicAlwaysFiltersToPublished(t *testing.T) {
	for _, query := range []string{
		"",
		"?status=pending",
		"?status=rejected",
	} {
		fake := newFakeStore()
		srv := newTestServer(fake)

		rec := doRequest(srv, http.MethodGet, "/api/v1/products/honey/reviews"+query, "", nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d for %q", rec.Code, query)
		}
		if got := fake.lastReviewFilter.Status; got != domain.ReviewPublished {
			t.Errorf("%q reached the store as status %q, want published", query, got)
		}
	}
}

func TestCreateReview(t *testing.T) {
	newAuthed := func(t *testing.T) (*fakeStore, http.Handler, *http.Cookie) {
		t.Helper()
		fake := newFakeStore()
		srv := newTestServer(fake)
		return fake, srv, loginAs(fake, domain.User{ID: 7, Role: "customer"})
	}

	t.Run("anonymous is 401", func(t *testing.T) {
		_, srv, _ := newAuthed(t)
		rec := doRequest(srv, http.MethodPost, "/api/v1/products/honey/reviews",
			`{"rating":5}`, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("stores a trimmed review and answers 201 pending", func(t *testing.T) {
		fake, srv, cookie := newAuthed(t)

		rec := doRequest(srv, http.MethodPost, "/api/v1/products/honey/reviews",
			`{"rating":5,"title":"  Lovely  ","body":"  Really good.  "}`, cookie)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		if fake.createdReview.Title != "Lovely" || fake.createdReview.Body != "Really good." {
			t.Errorf("stored untrimmed: %+v", fake.createdReview)
		}
		// The response says pending, so the UI can explain why the review is
		// not on the page yet instead of looking broken.
		if !strings.Contains(rec.Body.String(), `"status":"pending"`) {
			t.Errorf("body = %s", rec.Body.String())
		}
	})

	t.Run("a non-purchaser is 403, not 404", func(t *testing.T) {
		fake, srv, cookie := newAuthed(t)
		fake.reviewErr = domain.ErrNotPurchased

		rec := doRequest(srv, http.MethodPost, "/api/v1/products/honey/reviews",
			`{"rating":5}`, cookie)
		// The product exists and we know who is asking; what is missing is
		// standing, which is what 403 means.
		if rec.Code != http.StatusForbidden {
			t.Errorf("status = %d, want 403", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "not_purchased") {
			t.Errorf("body = %s", rec.Body.String())
		}
	})

	t.Run("a second review is 409", func(t *testing.T) {
		fake, srv, cookie := newAuthed(t)
		fake.reviewErr = domain.ErrAlreadyReviewed

		rec := doRequest(srv, http.MethodPost, "/api/v1/products/honey/reviews",
			`{"rating":1}`, cookie)
		if rec.Code != http.StatusConflict {
			t.Errorf("status = %d, want 409", rec.Code)
		}
	})

	t.Run("rating outside 1..5 is a field error", func(t *testing.T) {
		for _, body := range []string{`{"rating":0}`, `{"rating":6}`, `{}`} {
			_, srv, cookie := newAuthed(t)
			rec := doRequest(srv, http.MethodPost, "/api/v1/products/honey/reviews", body, cookie)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s → %d, want 400", body, rec.Code)
				continue
			}
			if !strings.Contains(rec.Body.String(), domain.ValidationRatingRange) {
				t.Errorf("%s → %s", body, rec.Body.String())
			}
		}
	})

	t.Run("an over-long body is rejected in runes, not bytes", func(t *testing.T) {
		_, srv, cookie := newAuthed(t)
		// Armenian: 1 rune, 2 bytes. A byte limit would let a third fewer
		// characters through here than in English.
		long, _ := json.Marshal(strings.Repeat("ա", domain.ReviewMaxBody+1))
		rec := doRequest(srv, http.MethodPost, "/api/v1/products/honey/reviews",
			`{"rating":5,"body":`+string(long)+`}`, cookie)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), domain.ValidationTooLong) {
			t.Errorf("body = %s", rec.Body.String())
		}

		// ...and exactly at the limit is fine.
		ok, _ := json.Marshal(strings.Repeat("ա", domain.ReviewMaxBody))
		_, srv2, cookie2 := newAuthed(t)
		rec = doRequest(srv2, http.MethodPost, "/api/v1/products/honey/reviews",
			`{"rating":5,"body":`+string(ok)+`}`, cookie2)
		if rec.Code != http.StatusCreated {
			t.Errorf("at the limit → %d, want 201", rec.Code)
		}
	})
}

// A shop that prints customers' email addresses beside their opinions has
// built a spam list and published it.
func TestReviewsNeverPublishEmailAddresses(t *testing.T) {
	fake := newFakeStore()
	fake.reviews = []domain.Review{
		{ID: 1, Rating: 5, Body: "Great", AuthorEmail: "alice.mccallister@example.com",
			Status: domain.ReviewPublished},
	}
	srv := newTestServer(fake)

	body := doRequest(srv, http.MethodGet, "/api/v1/products/honey/reviews", "", nil).Body.String()
	if strings.Contains(body, "@example.com") || strings.Contains(body, "mccallister") {
		t.Errorf("the public review list leaked an email address: %s", body)
	}
	// A display name is still there, and truncated — a long local part is
	// identifying on its own.
	if !strings.Contains(body, `"author":"alice.mccall…"`) {
		t.Errorf("expected a truncated display name: %s", body)
	}

	// The moderator DOES see it: judging whether a review is genuine
	// sometimes turns on who wrote it, and that response is behind
	// requireAdmin.
	admin := loginAs(fake, domain.User{ID: 1, Role: "admin"})
	adminBody := doRequest(srv, http.MethodGet, "/api/v1/admin/reviews", "", admin).Body.String()
	if !strings.Contains(adminBody, "alice.mccallister@example.com") {
		t.Errorf("the moderation queue hid the address it needs: %s", adminBody)
	}
}

func TestModerateReview(t *testing.T) {
	setup := func(t *testing.T) (*fakeStore, http.Handler, *http.Cookie) {
		t.Helper()
		fake := newFakeStore()
		return fake, newTestServer(fake), loginAs(fake, domain.User{ID: 1, Role: "admin"})
	}

	t.Run("publishes", func(t *testing.T) {
		fake, srv, cookie := setup(t)
		rec := doRequest(srv, http.MethodPatch, "/api/v1/admin/reviews/4",
			`{"status":"published"}`, cookie)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d: %s", rec.Code, rec.Body.String())
		}
		if fake.moderatedID != 4 || fake.moderatedStatus != domain.ReviewPublished {
			t.Errorf("store saw id=%d status=%q", fake.moderatedID, fake.moderatedStatus)
		}
	})

	t.Run("rejects a status outside the whitelist", func(t *testing.T) {
		// The CHECK constraint would catch it too, but as a 500-shaped
		// driver error rather than a field a form can point at.
		for _, body := range []string{
			`{"status":"approved"}`, `{"status":""}`,
			`{"status":"published; DROP TABLE reviews"}`,
		} {
			fake, srv, cookie := setup(t)
			rec := doRequest(srv, http.MethodPatch, "/api/v1/admin/reviews/4", body, cookie)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("%s → %d, want 400", body, rec.Code)
			}
			if fake.moderatedStatus != "" {
				t.Errorf("%s reached the store as %q", body, fake.moderatedStatus)
			}
		}
	})

	t.Run("requires an admin", func(t *testing.T) {
		fake := newFakeStore()
		srv := newTestServer(fake)
		customer := loginAs(fake, domain.User{ID: 2, Role: "customer"})

		if rec := doRequest(srv, http.MethodGet, "/api/v1/admin/reviews", "", nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("anonymous queue = %d, want 401", rec.Code)
		}
		if rec := doRequest(srv, http.MethodPatch, "/api/v1/admin/reviews/1",
			`{"status":"published"}`, customer); rec.Code != http.StatusForbidden {
			t.Errorf("customer moderation = %d, want 403", rec.Code)
		}
	})
}

// can_review saves the UI from reimplementing the verified-purchase rule.
func TestProductDetailCarriesCanReview(t *testing.T) {
	fake := newFakeStore()
	fake.products = []domain.Product{{ID: 1, Slug: "honey", Name: "Honey"}}
	fake.canReview = true
	srv := newTestServer(fake)

	// Anonymous: always false, and the store is never asked — an anonymous
	// visitor cannot review anything, so the round trip has a known answer.
	anon := doRequest(srv, http.MethodGet, "/api/v1/products/honey", "", nil).Body.String()
	if !strings.Contains(anon, `"can_review":false`) {
		t.Errorf("anonymous detail = %s", anon)
	}

	cookie := loginAs(fake, domain.User{ID: 7, Role: "customer"})
	authed := doRequest(srv, http.MethodGet, "/api/v1/products/honey", "", cookie).Body.String()
	if !strings.Contains(authed, `"can_review":true`) {
		t.Errorf("signed-in detail = %s", authed)
	}
}

// The aggregate is on the CARD too, because the design draws stars in the
// grid — which is the whole reason it is a stored column.
func TestListingCarriesTheRatingAggregate(t *testing.T) {
	fake := newFakeStore()
	fake.products = []domain.Product{{
		ID: 1, Slug: "honey", Name: "Honey",
		Rating: domain.RatingSummary{Average: 4.5, Count: 12},
	}}
	srv := newTestServer(fake)

	body := doRequest(srv, http.MethodGet, "/api/v1/products", "", nil).Body.String()
	if !strings.Contains(body, `"rating_avg":4.5`) || !strings.Contains(body, `"rating_count":12`) {
		t.Errorf("listing = %s", body)
	}
}
