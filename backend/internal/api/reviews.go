package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

type reviewResponse struct {
	ID        int64     `json:"id"`
	Rating    int       `json:"rating"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	// Only meaningful on the admin queue; the public list is all published.
	Status string `json:"status,omitempty"`
}

// displayName turns an email into something safe to print beside an opinion.
//
// A shop that publishes its customers' email addresses next to their reviews
// has built a spam list and handed it to anyone with a browser. The local
// part alone is enough to say "this is a person, and a different one from
// the review above", which is all the display needs — and it is truncated,
// because an address can be long enough to be identifying on its own.
func displayName(email string) string {
	local, _, found := strings.Cut(email, "@")
	if !found || local == "" {
		return "Customer"
	}
	if len([]rune(local)) > 12 {
		local = string([]rune(local)[:12]) + "…"
	}
	return local
}

func toReviewResponse(r domain.Review, includeStatus bool) reviewResponse {
	out := reviewResponse{
		ID: r.ID, Rating: r.Rating, Title: r.Title, Body: r.Body,
		Author: displayName(r.AuthorEmail), CreatedAt: r.CreatedAt,
	}
	if includeStatus {
		out.Status = r.Status
	}
	return out
}

// GET /products/{slug}/reviews — published reviews only, newest first.
func (s *Server) handleListReviews(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := domain.ReviewFilter{
		ProductSlug: chi.URLParam(r, "slug"),
		// Hardcoded, never taken from the query string: a public endpoint
		// that let a caller ask for `pending` would publish every review the
		// moderator has not looked at yet.
		Status:  domain.ReviewPublished,
		Page:    intQueryParam(q.Get("page"), 1),
		PerPage: intQueryParam(q.Get("per_page"), 10),
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 50 {
		filter.PerPage = 10
	}

	reviews, total, err := s.store.ListReviews(r.Context(), filter)
	if err != nil {
		s.log.Error("listing reviews", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	items := make([]reviewResponse, 0, len(reviews))
	for _, rv := range reviews {
		items = append(items, toReviewResponse(rv, false))
	}
	s.respondJSON(w, http.StatusOK, paginated[reviewResponse]{
		Items: items, Page: filter.Page, PerPage: filter.PerPage, Total: total,
	})
}

type createReviewRequest struct {
	Rating int    `json:"rating"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// POST /products/{slug}/reviews — login required, and a delivered order.
func (s *Server) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r.Context())
	if !ok {
		s.respondError(w, http.StatusUnauthorized, "unauthorized", "sign in to leave a review")
		return
	}

	var req createReviewRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}

	review := domain.TrimReview(domain.Review{
		UserID: user.ID, Rating: req.Rating, Title: req.Title, Body: req.Body,
	})
	if fields := domain.ValidateReview(review); len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	err := s.store.CreateReview(r.Context(), &review, chi.URLParam(r, "slug"))
	switch {
	case errors.Is(err, domain.ErrNotFound):
		s.respondError(w, http.StatusNotFound, "not_found", "no such product")
		return
	case errors.Is(err, domain.ErrNotPurchased):
		// 403, not 404: the product exists and we know who is asking. What
		// is missing is standing, which is precisely what 403 means.
		s.respondError(w, http.StatusForbidden, "not_purchased",
			"you can review a product once it has been delivered to you")
		return
	case errors.Is(err, domain.ErrAlreadyReviewed):
		s.respondError(w, http.StatusConflict, "already_reviewed",
			"you have already reviewed this product")
		return
	case err != nil:
		s.log.Error("creating review", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// 201 with the review as stored — including `status: "pending"`, so the
	// UI can say "thanks, it will appear once we have read it" rather than
	// showing a review that is not public yet and letting the reader wonder
	// why nobody else can see it.
	s.respondJSON(w, http.StatusCreated, toReviewResponse(review, true))
}

// GET /admin/reviews?status=pending — the moderation queue.
func (s *Server) handleAdminListReviews(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	if status != "" && !domain.ValidReviewStatus(status) {
		s.respondValidationError(w, map[string]string{"status": domain.ValidationInvalidStatus})
		return
	}

	filter := domain.ReviewFilter{
		ProductSlug: q.Get("product"),
		Status:      status, // empty = every status, which the queue wants
		Page:        intQueryParam(q.Get("page"), 1),
		PerPage:     intQueryParam(q.Get("per_page"), 50),
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 50
	}

	reviews, total, err := s.store.ListReviews(r.Context(), filter)
	if err != nil {
		s.log.Error("admin listing reviews", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	items := make([]adminReviewResponse, 0, len(reviews))
	for _, rv := range reviews {
		items = append(items, adminReviewResponse{
			reviewResponse: toReviewResponse(rv, true),
			ProductID:      rv.ProductID,
			// The moderator DOES see the address: deciding whether a review
			// is genuine sometimes turns on who wrote it, and this response
			// is behind requireAdmin.
			Email: rv.AuthorEmail,
		})
	}
	s.respondJSON(w, http.StatusOK, paginated[adminReviewResponse]{
		Items: items, Page: filter.Page, PerPage: filter.PerPage, Total: total,
	})
}

type adminReviewResponse struct {
	reviewResponse
	ProductID int64  `json:"product_id"`
	Email     string `json:"email"`
}

type moderateReviewRequest struct {
	Status string `json:"status"`
}

// PATCH /admin/reviews/{id} — publish or reject.
func (s *Server) handleModerateReview(w http.ResponseWriter, r *http.Request) {
	id, ok := s.pathID(w, r, "id")
	if !ok {
		return
	}

	var req moderateReviewRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	// Whitelisted before it reaches SQL, like every other caller-supplied
	// enum in this codebase. The CHECK constraint would catch it too, but as
	// a 500-shaped driver error rather than a field the form can point at.
	if !domain.ValidReviewStatus(req.Status) {
		s.respondValidationError(w, map[string]string{"status": domain.ValidationInvalidStatus})
		return
	}

	review, err := s.store.UpdateReviewStatus(r.Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such review")
			return
		}
		s.log.Error("moderating review", "review_id", id, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toReviewResponse(review, true))
}
