package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

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
	Slug string `json:"slug"`
	// Name is the English name and is required — every other language falls
	// back to it, so it is the one translation that cannot be missing.
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	// Translations is optional and holds only the non-default languages:
	// {"hy": "Մեղր", "ru": "Мёд"}. Leaving a language out is normal and means
	// "show English there"; an "en" key is rejected, since that value already
	// has a home in Name.
	//
	// Added rather than replacing Name so the change is backward compatible —
	// every existing client and Postman request keeps working untouched.
	Translations map[string]string `json:"translations"`
}

// parseLocaleMap converts the wire's string keys into domain.Locale keys.
// Unknown keys are preserved as-is rather than dropped, so validation can
// report them as `translations.xx` instead of silently ignoring a typo.
func parseLocaleMap[T any](in map[string]T) map[domain.Locale]T {
	if len(in) == 0 {
		return nil
	}
	out := make(map[domain.Locale]T, len(in))
	for k, v := range in {
		out[domain.Locale(k)] = v
	}
	return out
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
	cats, err := s.store.ListCategories(r.Context(), localeFromContext(r.Context()))
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

	translations := parseLocaleMap(req.Translations)

	fields := domain.ValidateCategory(req.Slug, req.Name)
	for k, v := range domain.ValidateCategoryTranslations(translations) {
		fields[k] = v
	}
	if len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	cat := domain.Category{
		Slug:         req.Slug,
		Name:         req.Name,
		SortOrder:    req.SortOrder,
		Translations: translations,
	}
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

// ── F2: category management (decision #95) ────────────────────────────────

// adminCategoryResponse is the EDITOR's shape: name is raw English and the
// non-default translations ride along, because a form that cannot see what
// it is editing cannot edit it. The public categoryResponse stays
// locale-resolved and translation-free.
type adminCategoryResponse struct {
	ID           int64             `json:"id"`
	Slug         string            `json:"slug"`
	Name         string            `json:"name"`
	SortOrder    int               `json:"sort_order"`
	Translations map[string]string `json:"translations"`
}

// GET /admin/categories — the editor's list.
func (s *Server) handleAdminListCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.store.AdminCategories(r.Context())
	if err != nil {
		s.log.Error("listing admin categories", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp := make([]adminCategoryResponse, 0, len(cats))
	for _, c := range cats {
		out := adminCategoryResponse{
			ID: c.ID, Slug: c.Slug, Name: c.Name, SortOrder: c.SortOrder,
			Translations: make(map[string]string, len(c.Translations)),
		}
		for locale, name := range c.Translations {
			out.Translations[string(locale)] = name
		}
		resp = append(resp, out)
	}
	s.respondJSON(w, http.StatusOK, resp)
}

// PUT /admin/categories/{id} — whole-value update, the create request's
// shape re-applied: same validation, same slug_taken vocabulary.
func (s *Server) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	catID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "category id must be a number")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req createCategoryRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	translations := parseLocaleMap(req.Translations)
	fields := domain.ValidateCategory(req.Slug, req.Name)
	for k, v := range domain.ValidateCategoryTranslations(translations) {
		fields[k] = v
	}
	if len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	cat := domain.Category{
		ID: catID, Slug: req.Slug, Name: req.Name,
		SortOrder: req.SortOrder, Translations: translations,
	}
	if err := s.store.UpdateCategory(r.Context(), &cat); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			s.respondError(w, http.StatusNotFound, "not_found", "no such category")
		case errors.Is(err, domain.ErrSlugTaken):
			s.respondError(w, http.StatusConflict, "slug_taken", "a category with this slug already exists")
		default:
			s.log.Error("updating category", "id", catID, "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	s.respondJSON(w, http.StatusOK, toCategoryResponse(cat))
}

// DELETE /admin/categories/{id} — empty categories only. The refusal for a
// category holding products is the SCHEMA's (ON DELETE RESTRICT),
// surfaced as 409 category_in_use; the admin moves or retires the
// products first, and the UI says so.
func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	catID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "category id must be a number")
		return
	}
	if err := s.store.DeleteCategory(r.Context(), catID); err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			s.respondError(w, http.StatusNotFound, "not_found", "no such category")
		case errors.Is(err, domain.ErrCategoryInUse):
			s.respondError(w, http.StatusConflict, "category_in_use",
				"this category still has products — move or retire them first")
		default:
			s.log.Error("deleting category", "id", catID, "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type reorderCategoriesRequest struct {
	IDs []int64 `json:"ids"`
}

// PUT /admin/categories/order — the ordered id list becomes sort_order by
// position. The UI sends its WHOLE list after every move; an unknown id
// aborts the transaction (404), so a stale list cannot half-apply.
// Registered before the {id} routes match: chi prefers the static
// segment, so "order" is never parsed as an id.
func (s *Server) handleReorderCategories(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req reorderCategoriesRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	if len(req.IDs) == 0 {
		s.respondValidationError(w, map[string]string{"ids": "required"})
		return
	}
	if err := s.store.ReorderCategories(r.Context(), req.IDs); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "an id in the list matches no category")
			return
		}
		s.log.Error("reordering categories", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
