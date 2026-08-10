package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

type cartItemResponse struct {
	VariantID      int64  `json:"variant_id"`
	ProductName    string `json:"product_name"`
	ProductSlug    string `json:"product_slug"`
	Label          string `json:"label"`
	PriceMinor     int64  `json:"price_minor"`
	StockQty       int    `json:"stock_qty"`
	Qty            int    `json:"qty"`
	LineTotalMinor int64  `json:"line_total_minor"`
}

type cartResponse struct {
	Items      []cartItemResponse `json:"items"`
	TotalMinor int64              `json:"total_minor"`
}

type setCartItemRequest struct {
	VariantID int64 `json:"variant_id"`
	Qty       int   `json:"qty"`
}

func toCartResponse(items []domain.CartItem) cartResponse {
	resp := cartResponse{Items: make([]cartItemResponse, 0, len(items))}
	for _, it := range items {
		resp.Items = append(resp.Items, cartItemResponse{
			VariantID:      it.VariantID,
			ProductName:    it.ProductName,
			ProductSlug:    it.ProductSlug,
			Label:          it.Label,
			PriceMinor:     it.PriceMinor,
			StockQty:       it.StockQty,
			Qty:            it.Qty,
			LineTotalMinor: it.LineTotalMinor(),
		})
	}
	resp.TotalMinor = domain.CartTotalMinor(items)
	return resp
}

func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context()) // requireUser guarantees presence

	items, err := s.store.GetCart(r.Context(), user.ID, localeFromContext(r.Context()))
	if err != nil {
		s.log.Error("getting cart", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toCartResponse(items))
}

// PUT /cart/items — idempotent "set quantity" semantics: sending the same
// request twice leaves the same state.
func (s *Server) handleSetCartItem(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req setCartItemRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	fields := make(map[string]string)
	if req.VariantID <= 0 {
		fields["variant_id"] = "required"
	}
	if req.Qty < 1 || req.Qty > 99 {
		fields["qty"] = "must be between 1 and 99"
	}
	if len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	if err := s.store.SetCartItem(r.Context(), user.ID, req.VariantID, req.Qty); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such product variant")
			return
		}
		s.log.Error("setting cart item", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	items, err := s.store.GetCart(r.Context(), user.ID, localeFromContext(r.Context()))
	if err != nil {
		s.log.Error("reloading cart", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toCartResponse(items))
}

func (s *Server) handleDeleteCartItem(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	variantID, err := strconv.ParseInt(chi.URLParam(r, "variantID"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "variant id must be a number")
		return
	}

	if err := s.store.DeleteCartItem(r.Context(), user.ID, variantID); err != nil {
		s.log.Error("deleting cart item", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
