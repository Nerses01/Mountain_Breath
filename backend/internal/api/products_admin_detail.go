package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The admin endpoints for a product page's editorial half (E3).
//
// All four are PUT (or DELETE), never PATCH: each replaces a whole collection
// with what the form is showing, which is idempotent — submitting twice
// leaves the same state. A PATCH would imply a partial merge the store does
// not do.

// pathID reads a numeric URL parameter, answering 400 rather than letting a
// non-numeric id reach the store as a zero.
func (s *Server) pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil || id <= 0 {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "invalid "+name)
		return 0, false
	}
	return id, true
}

// decodeJSON reads a capped, strict JSON body. Shared by the handlers below
// so the 1 MB cap and DisallowUnknownFields cannot be forgotten on one of
// them — a typo'd field name should be a 400, not silence.
func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json",
			"request body is not valid JSON: "+err.Error())
		return false
	}
	return true
}

// --- Images ------------------------------------------------------------

type imageInputRequest struct {
	ID        int64 `json:"id"`
	SortOrder int   `json:"sort_order"`
	IsPrimary bool  `json:"is_primary"`
	// Alt text per locale, INCLUDING "en" — unlike product translations,
	// where English lives in the parent field. There is no parent field for
	// an image's alt, so all three languages are peers here.
	Alt map[string]string `json:"alt"`
}

type saveImagesRequest struct {
	Images []imageInputRequest `json:"images"`
}

// PUT /admin/products/{id}/images — reorder, set the hero, edit alt text.
func (s *Server) handleSaveProductImages(w http.ResponseWriter, r *http.Request) {
	productID, ok := s.pathID(w, r, "id")
	if !ok {
		return
	}
	var req saveImagesRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}

	// Exactly one hero, checked before the store so the failure is a 400 the
	// form can attach to a field rather than a constraint violation.
	primaries := 0
	for _, img := range req.Images {
		if img.IsPrimary {
			primaries++
		}
	}
	if len(req.Images) > 0 && primaries != 1 {
		s.respondValidationError(w, map[string]string{"images": domain.ValidationOnePrimary})
		return
	}

	images := make([]domain.ProductImage, 0, len(req.Images))
	alts := make(map[int64]map[domain.Locale]string, len(req.Images))
	for i, img := range req.Images {
		// Position comes from the ARRAY ORDER, not from the client's
		// sort_order: the list the admin sees is the list they dragged, and
		// trusting a separate number invites the two to disagree.
		images = append(images, domain.ProductImage{
			ID: img.ID, SortOrder: i, IsPrimary: img.IsPrimary,
		})
		alts[img.ID] = parseLocaleMap(img.Alt)
	}

	if err := s.store.SaveProductImages(r.Context(), productID, images, alts); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such product or image")
			return
		}
		s.log.Error("saving product images", "product_id", productID, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /admin/products/{id}/images/{imageID}
func (s *Server) handleDeleteProductImage(w http.ResponseWriter, r *http.Request) {
	productID, ok := s.pathID(w, r, "id")
	if !ok {
		return
	}
	imageID, ok := s.pathID(w, r, "imageID")
	if !ok {
		return
	}

	if err := s.store.DeleteProductImage(r.Context(), productID, imageID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such image")
			return
		}
		s.log.Error("deleting product image", "image_id", imageID, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Editorial ---------------------------------------------------------

type highlightRequest struct {
	Text string `json:"text"`
}

type usageCardRequest struct {
	Kicker string `json:"kicker"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

type editorialRequest struct {
	Highlights []highlightRequest `json:"highlights"`
	UsageCards []usageCardRequest `json:"usage_cards"`
}

// The wire shape is keyed by locale — {"en": {...}, "hy": {...}} — mirroring
// storage exactly (migration 000012), where the ROW is per-language. "en" is
// a legitimate key here, unlike in product translations, because there is no
// parent field holding the English copy.
type saveEditorialRequest struct {
	Content map[string]editorialRequest `json:"content"`
}

// PUT /admin/products/{id}/editorial
func (s *Server) handleSaveProductEditorial(w http.ResponseWriter, r *http.Request) {
	productID, ok := s.pathID(w, r, "id")
	if !ok {
		return
	}
	var req saveEditorialRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}

	fields := make(map[string]string)
	byLocale := make(map[domain.Locale]domain.ProductEditorial, len(req.Content))
	for rawLocale, content := range req.Content {
		locale, valid := domain.ParseLocale(rawLocale)
		if !valid {
			fields["content."+rawLocale] = domain.ValidationLocaleUnsupported
			continue
		}

		editorial := domain.ProductEditorial{
			Highlights: make([]domain.ProductHighlight, 0, len(content.Highlights)),
			UsageCards: make([]domain.ProductUsageCard, 0, len(content.UsageCards)),
		}
		for _, h := range content.Highlights {
			editorial.Highlights = append(editorial.Highlights, domain.ProductHighlight{Text: h.Text})
		}
		for _, c := range content.UsageCards {
			editorial.UsageCards = append(editorial.UsageCards, domain.ProductUsageCard{
				Kicker: c.Kicker, Title: c.Title, Body: c.Body,
			})
		}
		for k, v := range domain.ValidateEditorial("content."+rawLocale, editorial) {
			fields[k] = v
		}
		byLocale[locale] = editorial
	}
	if len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	if err := s.store.SaveProductEditorial(r.Context(), productID, byLocale); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such product")
			return
		}
		s.log.Error("saving product editorial", "product_id", productID, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Related -----------------------------------------------------------

type saveRelatedRequest struct {
	// Ordered: the array's order IS the display order.
	RelatedIDs []int64 `json:"related_ids"`
}

// PUT /admin/products/{id}/related
func (s *Server) handleSaveProductRelated(w http.ResponseWriter, r *http.Request) {
	productID, ok := s.pathID(w, r, "id")
	if !ok {
		return
	}
	var req saveRelatedRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}

	if err := s.store.SaveProductRelated(r.Context(), productID, req.RelatedIDs); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such product")
			return
		}
		s.log.Error("saving related products", "product_id", productID, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
