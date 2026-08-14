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
	VariantID   int64  `json:"variant_id"`
	ProductName string `json:"product_name"`
	ProductSlug string `json:"product_slug"`
	Label       string `json:"label"`
	StockQty    int    `json:"stock_qty"`
	Qty         int    `json:"qty"`

	// Denominated in the response's `currency`; `prices` and `line_totals`
	// carry the same line in every market for the design's second line.
	PriceMinor     int64                     `json:"price_minor"`
	LineTotalMinor int64                     `json:"line_total_minor"`
	Prices         map[domain.Currency]int64 `json:"prices"`
	LineTotals     map[domain.Currency]int64 `json:"line_totals"`
}

type cartResponse struct {
	Items    []cartItemResponse `json:"items"`
	Currency domain.Currency    `json:"currency"`

	// E6 renamed the MEANING of total_minor: it is now subtotal + shipping,
	// because that is what "Total" on the design's summary card says, and a
	// client rendering "Total" next to a number that silently omitted
	// shipping would be lying one screen before checkout told the truth.
	// subtotal_minor is the old sum-of-lines.
	SubtotalMinor int64 `json:"subtotal_minor"`
	ShippingMinor int64 `json:"shipping_minor"`
	TotalMinor    int64 `json:"total_minor"`

	// True when any line is a chilled product — the flag behind the
	// "Chilled shipping" label, so the client names the fee correctly
	// without re-deriving the rule.
	HasColdChain bool `json:"has_cold_chain"`

	// The same three figures in every market, summed INDEPENDENTLY, never
	// converted — see domain.Money.AddTo for why that distinction is the
	// whole point. A market any line cannot be priced in is absent rather
	// than understated.
	Subtotals map[domain.Currency]int64 `json:"subtotals"`
	Shipping  map[domain.Currency]int64 `json:"shipping"`
	Totals    map[domain.Currency]int64 `json:"totals"`
}

type setCartItemRequest struct {
	VariantID int64 `json:"variant_id"`
	Qty       int   `json:"qty"`
}

func toCartResponse(items []domain.CartItem, currency domain.Currency,
	rates map[domain.Currency]domain.ShippingRate) cartResponse {

	resp := cartResponse{
		Items:    make([]cartItemResponse, 0, len(items)),
		Currency: currency,
	}
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
			Prices:         it.Prices,
			LineTotals:     it.LineTotals(),
		})
	}

	// One arithmetic for the number the customer reads and the number the
	// checkout will charge: QuoteCart here, ComputeTotals in CreateOrder,
	// both built on the same ShippingFor.
	quote := domain.QuoteCart(items, rates)
	resp.HasColdChain = quote.HasColdChain
	resp.Subtotals = quote.SubtotalMinor
	resp.Shipping = quote.ShippingMinor
	resp.Totals = quote.TotalMinor
	resp.SubtotalMinor = quote.SubtotalMinor[currency]
	resp.ShippingMinor = quote.ShippingMinor[currency]
	resp.TotalMinor = quote.TotalMinor[currency]
	return resp
}

func (s *Server) handleGetCart(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context()) // requireUser guarantees presence

	view := viewFromContext(r.Context())
	items, err := s.store.GetCart(r.Context(), user.ID, view)
	if err != nil {
		s.log.Error("getting cart", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	rates, err := s.store.ShippingRates(r.Context())
	if err != nil {
		s.log.Error("loading shipping rates", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toCartResponse(items, view.EffectiveCurrency(), rates))
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

	view := viewFromContext(r.Context())
	items, err := s.store.GetCart(r.Context(), user.ID, view)
	if err != nil {
		s.log.Error("reloading cart", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	rates, err := s.store.ShippingRates(r.Context())
	if err != nil {
		s.log.Error("loading shipping rates", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toCartResponse(items, view.EffectiveCurrency(), rates))
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
