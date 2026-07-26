package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// DTOs: the JSON shapes of the API, kept separate from domain types so the
// wire format can evolve independently of business logic.

type categoryResponse struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
}

type createCategoryRequest struct {
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

func toCategoryResponse(c domain.Category) categoryResponse {
	return categoryResponse{
		ID:        c.ID,
		Slug:      c.Slug,
		Name:      c.Name,
		SortOrder: c.SortOrder,
		CreatedAt: c.CreatedAt,
	}
}

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.store.ListCategories(r.Context())
	if err != nil {
		s.log.Error("listing categories", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	resp := make([]categoryResponse, 0, len(cats))
	for _, c := range cats {
		resp = append(resp, toCategoryResponse(c))
	}
	s.respondJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	// Cap the body at 1 MB: without this, a client could stream gigabytes
	// into our JSON decoder.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // typo'd field names become errors, not silence

	var req createCategoryRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	if fields := domain.ValidateCategory(req.Slug, req.Name); len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	cat := domain.Category{Slug: req.Slug, Name: req.Name, SortOrder: req.SortOrder}
	if err := s.store.CreateCategory(r.Context(), &cat); err != nil {
		if errors.Is(err, domain.ErrSlugTaken) {
			s.respondError(w, http.StatusConflict, "slug_taken", "a category with this slug already exists")
			return
		}
		s.log.Error("creating category", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	s.respondJSON(w, http.StatusCreated, toCategoryResponse(cat))
}
