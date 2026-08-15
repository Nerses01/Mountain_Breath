package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The wishlist (E8): the hearts across screens 01–04, and the design's
// "Save for later" on the cart.
//
// The response reuses the product CARD shape — a wishlist is a shelf of
// saved cards, and inventing a leaner "wishlist item" DTO would just be a
// second projection of the same product for the frontend to maintain. The
// heart's on/off state everywhere else in the UI derives from this one
// list, client-side: six products make membership a set lookup, not an
// endpoint.

func (s *Server) handleListWishlist(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	view := viewFromContext(r.Context())
	products, err := s.store.ListWishlist(r.Context(), user.ID, view)
	if err != nil {
		s.log.Error("listing wishlist", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp := make([]productResponse, 0, len(products))
	for _, p := range products {
		resp = append(resp, toProductResponse(p, view.EffectiveCurrency()))
	}
	s.respondJSON(w, http.StatusOK, resp)
}

// PUT and DELETE with set-semantics, like the cart: the URL names the
// desired STATE ("product 5 is hearted"), so repeating a request changes
// nothing and the heart button needs no read-before-write.
func (s *Server) handleAddWishlistItem(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "product id must be a number")
		return
	}

	if err := s.store.AddWishlistItem(r.Context(), user.ID, productID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such product")
			return
		}
		s.log.Error("adding wishlist item", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleRemoveWishlistItem(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	productID, err := strconv.ParseInt(chi.URLParam(r, "productID"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "product id must be a number")
		return
	}

	if err := s.store.RemoveWishlistItem(r.Context(), user.ID, productID); err != nil {
		s.log.Error("removing wishlist item", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type saveForLaterRequest struct {
	VariantID int64 `json:"variant_id"`
}

// POST /wishlist/save-for-later — the cart's "not today, but don't lose it"
// move. POST rather than PUT because it is a TRANSFER, not a state to
// idempotently assert: the second identical request finds no cart line and
// honestly 404s.
func (s *Server) handleSaveForLater(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req saveForLaterRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	if req.VariantID <= 0 {
		s.respondValidationError(w, map[string]string{"variant_id": domain.ValidationRequired})
		return
	}

	if err := s.store.SaveForLater(r.Context(), user.ID, req.VariantID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such line in your cart")
			return
		}
		s.log.Error("saving for later", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
