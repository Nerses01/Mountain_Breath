package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// E7: the checkout preview and the promo box.
//
// POST /checkout/preview is THE money endpoint of the storefront: every
// figure the cart page and the checkout sidebar render comes from here, and
// the order that POST /orders eventually creates is priced by the same
// domain.Price call with the same inputs read inside its transaction. The
// client's remaining arithmetic is formatting.

type upsellResponse struct {
	VariantID  int64  `json:"variant_id"`
	Slug       string `json:"slug"`
	Name       string `json:"name"`
	PriceMinor int64  `json:"price_minor"`
}

type previewResponse struct {
	Currency domain.Currency `json:"currency"`

	SubtotalMinor       int64 `json:"subtotal_minor"`
	ShippingMinor       int64 `json:"shipping_minor"`
	MemberDiscountMinor int64 `json:"member_discount_minor"`
	PromoDiscountMinor  int64 `json:"promo_discount_minor"`
	DiscountMinor       int64 `json:"discount_minor"`
	TaxMinor            int64 `json:"tax_minor"`
	TotalMinor          int64 `json:"total_minor"`

	HasColdChain       bool `json:"has_cold_chain"`
	FirstDeliveryFree  bool `json:"first_delivery_free"`
	BaseShippingWaived bool `json:"base_shipping_waived"`

	// The progress bar's two numbers. Both absent once the base is waived
	// (or the market has no threshold): a bar with nothing to count toward
	// is not drawn, so it is not sent. The client divides one by the other
	// for the bar's width — display math, not money math.
	FreeShippingThresholdMinor *int64 `json:"free_shipping_threshold_minor,omitempty"`
	FreeShippingRemainingMinor *int64 `json:"free_shipping_remaining_minor,omitempty"`

	// The banner's call to action: one product that would close the gap.
	Upsell *upsellResponse `json:"upsell,omitempty"`

	// The attached code, even when it currently cannot apply — PromoIssue
	// says why (a validation code the client's catalogue renders), so the
	// box can complain about the code BY NAME instead of silently dropping
	// a discount the customer thinks they have.
	PromoCode  string `json:"promo_code,omitempty"`
	PromoKind  string `json:"promo_kind,omitempty"`
	PromoIssue string `json:"promo_issue,omitempty"`

	// The grand total in every market, for the design's muted second line —
	// same intersection rule as the cart: a market this basket (or its
	// promo) cannot be honestly priced in is absent, never approximated.
	Totals map[domain.Currency]int64 `json:"totals"`
}

// buildPreview gathers what pricing depends on and lets domain.Price decide.
// Shared by the preview endpoint and both promo handlers, which answer with
// a fresh preview — applying a code and THEN asking what things cost would
// be two round trips racing each other.
func (s *Server) buildPreview(r *http.Request, userID int64) (previewResponse, error) {
	ctx := r.Context()
	view := viewFromContext(ctx)
	primary := view.EffectiveCurrency()

	items, err := s.store.GetCart(ctx, userID, view)
	if err != nil {
		return previewResponse{}, err
	}

	resp := previewResponse{Currency: primary, Totals: map[domain.Currency]int64{}}
	if len(items) == 0 {
		// An empty basket has no shipping, no discounts and no bar — zeros,
		// not an error: the cart page renders its empty state, and a
		// preview that 4xx'd on emptiness would make that state an error.
		return resp, nil
	}

	rates, err := s.store.ShippingRates(ctx)
	if err != nil {
		return previewResponse{}, err
	}
	promo, err := s.store.CartPromoForUser(ctx, userID)
	if err != nil {
		return previewResponse{}, err
	}
	prior, err := s.store.PriorOrders(ctx, userID)
	if err != nil {
		return previewResponse{}, err
	}

	subtotals := domain.CartTotals(items)
	hasColdChain := false
	for _, it := range items {
		if it.IsColdChain {
			hasColdChain = true
		}
	}
	now := time.Now()

	price := func(c domain.Currency) (domain.Breakdown, bool) {
		rate, ok := rates[c]
		if !ok {
			return domain.Breakdown{}, false
		}
		subtotal, ok := subtotals[c]
		if !ok {
			return domain.Breakdown{}, false
		}
		return domain.Price(domain.PriceInput{
			Currency: c, SubtotalMinor: subtotal, HasColdChain: hasColdChain,
			Rate: rate, PriorOrders: prior, Promo: promo, Now: now,
		}), true
	}

	b, ok := price(primary)
	if !ok {
		// The shopper's market cannot be quoted (no rate row, or a line
		// with no price there). Same honesty as the cart: the figures for
		// this market are absent rather than zero — the client shows what
		// it can and checkout will refuse anyway.
		return resp, nil
	}

	resp.SubtotalMinor = b.SubtotalMinor
	resp.ShippingMinor = b.ShippingMinor
	resp.MemberDiscountMinor = b.MemberDiscountMinor
	resp.PromoDiscountMinor = b.PromoDiscountMinor
	resp.DiscountMinor = b.DiscountMinor
	resp.TaxMinor = b.TaxMinor
	resp.TotalMinor = b.TotalMinor
	resp.HasColdChain = hasColdChain
	resp.FirstDeliveryFree = b.FirstDeliveryFree
	resp.BaseShippingWaived = b.BaseShippingWaived
	resp.PromoCode = b.PromoCode
	resp.PromoIssue = b.PromoIssue
	if promo != nil {
		resp.PromoKind = promo.Kind
	}
	if b.RemainingForFreeShippingMinor != nil {
		resp.FreeShippingRemainingMinor = b.RemainingForFreeShippingMinor
		resp.FreeShippingThresholdMinor = rates[primary].FreeOverMinor

		upsell, err := s.store.UpsellForGap(ctx, view, *b.RemainingForFreeShippingMinor)
		if err != nil {
			return previewResponse{}, err
		}
		if upsell != nil {
			resp.Upsell = &upsellResponse{
				VariantID: upsell.VariantID, Slug: upsell.Slug,
				Name: upsell.Name, PriceMinor: upsell.PriceMinor,
			}
		}
	}

	// The dual-total map. A market goes in only when the promo participated
	// there exactly as it did in the primary — a secondary total that
	// silently dropped the discount would be a wrong number that looks
	// right, the exact failure CartTotals' intersection rule exists to
	// prevent.
	for c := range subtotals {
		alt, ok := price(c)
		if !ok || alt.PromoIssue != b.PromoIssue {
			continue
		}
		resp.Totals[c] = alt.TotalMinor
	}
	return resp, nil
}

// POST /checkout/preview — no body: everything it prices is server-side
// state (the cart, the applied code, the customer's history) plus the
// negotiated view. POST rather than GET because the answer is a computation
// about YOUR session this instant — private, uncacheable, and shaped by
// cookies — not an addressable resource.
func (s *Server) handleCheckoutPreview(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	resp, err := s.buildPreview(r, user.ID)
	if err != nil {
		s.log.Error("building checkout preview", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, resp)
}

type applyPromoRequest struct {
	Code string `json:"code"`
}

// POST /cart/promo — attach a code to the cart. Every way a code can be
// wrong is a FIELD error on "promo_code" (the same envelope as any form),
// because the promo box is a form input and its failures belong under it.
func (s *Server) handleApplyPromo(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req applyPromoRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	code := domain.NormalizePromoCode(req.Code)
	if code == "" {
		s.respondValidationError(w, map[string]string{"promo_code": domain.ValidationRequired})
		return
	}

	promo, err := s.store.PromoForUser(r.Context(), code, user.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondValidationError(w, map[string]string{"promo_code": domain.ValidationPromoUnknown})
			return
		}
		s.log.Error("looking up promo code", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// Judge the code against THIS basket before attaching it: "applied!"
	// followed by a preview complaining is worse feedback than the box
	// saying immediately why not. The same Issue() runs again on every
	// preview and once more under lock at checkout — apply-time validity is
	// a courtesy, checkout-time validity is the rule.
	view := viewFromContext(r.Context())
	currency := view.EffectiveCurrency()
	items, err := s.store.GetCart(r.Context(), user.ID, view)
	if err != nil {
		s.log.Error("loading cart for promo apply", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	subtotal := domain.CartTotals(items)[currency]
	if issue := promo.Issue(time.Now(), currency, subtotal); issue != "" {
		s.respondValidationError(w, map[string]string{"promo_code": issue})
		return
	}

	if err := s.store.SetCartPromo(r.Context(), user.ID, promo.ID); err != nil {
		s.log.Error("setting cart promo", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	resp, err := s.buildPreview(r, user.ID)
	if err != nil {
		s.log.Error("building preview after promo apply", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, resp)
}

// DELETE /cart/promo — detach the code. Answers with a fresh preview for
// the same reason apply does: the caller's next question is always "so what
// does it cost now".
func (s *Server) handleRemovePromo(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	if err := s.store.ClearCartPromo(r.Context(), user.ID); err != nil {
		s.log.Error("clearing cart promo", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp, err := s.buildPreview(r, user.ID)
	if err != nil {
		s.log.Error("building preview after promo remove", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, resp)
}
