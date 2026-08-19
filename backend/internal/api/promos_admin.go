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

// F2 (decision #94): the admin's promo CRUD — E7 shipped seed-only codes
// and wrote "revisit when three codes stop being enough". Note there is no
// DELETE: redemption history hangs off the code, so the off switch is
// `active`, flipped through the same whole-value update as everything else.

type promoValuePayload struct {
	AmountMinor      *int64 `json:"amount_minor,omitempty"`
	MinSubtotalMinor *int64 `json:"min_subtotal_minor,omitempty"`
}

type adminPromoResponse struct {
	ID      int64  `json:"id"`
	Code    string `json:"code"`
	Kind    string `json:"kind"`
	Percent *int   `json:"percent,omitempty"`

	StartsAt *time.Time `json:"starts_at,omitempty"`
	EndsAt   *time.Time `json:"ends_at,omitempty"`

	MaxRedemptions *int `json:"max_redemptions,omitempty"`
	Active         bool `json:"active"`
	// The live fact the admin table shows next to the cap. Read-only:
	// usage is history, not a property the form edits.
	Redemptions int `json:"redemptions"`

	Values map[string]promoValuePayload `json:"values"`
}

func toAdminPromoResponse(p domain.Promo) adminPromoResponse {
	resp := adminPromoResponse{
		ID: p.ID, Code: p.Code, Kind: p.Kind,
		StartsAt: p.StartsAt, EndsAt: p.EndsAt,
		MaxRedemptions: p.MaxRedemptions, Active: p.Active,
		Redemptions: p.Redemptions,
		Values:      make(map[string]promoValuePayload, len(p.Values)),
	}
	// Percent is meaningful only for the percent kind (the biconditional);
	// omitempty on a plain int would also hide a real value, so a pointer
	// carries "absent" honestly.
	if p.Kind == domain.PromoPercent {
		pct := p.Percent
		resp.Percent = &pct
	}
	for currency, v := range p.Values {
		resp.Values[string(currency)] = promoValuePayload{
			AmountMinor: v.AmountMinor, MinSubtotalMinor: v.MinSubtotalMinor,
		}
	}
	return resp
}

// promoRequest is the write shape for both create and update — whole-value,
// like the variant editor: the form shows every field, so it sends every
// field, and a currency left out of `values` is removed.
type promoRequest struct {
	Code    string `json:"code"`
	Kind    string `json:"kind"`
	Percent *int   `json:"percent"`

	StartsAt *time.Time `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`

	MaxRedemptions *int `json:"max_redemptions"`
	Active         bool `json:"active"`

	Values map[string]promoValuePayload `json:"values"`
}

func (r promoRequest) toDomain() domain.PromoInput {
	in := domain.PromoInput{
		Code: r.Code, Kind: r.Kind, Percent: r.Percent,
		StartsAt: r.StartsAt, EndsAt: r.EndsAt,
		MaxRedemptions: r.MaxRedemptions, Active: r.Active,
		Values: make(map[domain.Currency]domain.PromoValue, len(r.Values)),
	}
	// Keys pass through as typed; ValidatePromoInput judges whether each
	// names a currency the shop serves — the handler does not pre-filter.
	for currency, v := range r.Values {
		in.Values[domain.Currency(currency)] = domain.PromoValue{
			AmountMinor: v.AmountMinor, MinSubtotalMinor: v.MinSubtotalMinor,
		}
	}
	return in
}

// GET /admin/promos — every code, newest first.
func (s *Server) handleAdminListPromos(w http.ResponseWriter, r *http.Request) {
	promos, err := s.store.ListPromos(r.Context())
	if err != nil {
		s.log.Error("listing promos", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp := make([]adminPromoResponse, 0, len(promos))
	for _, p := range promos {
		resp = append(resp, toAdminPromoResponse(p))
	}
	s.respondJSON(w, http.StatusOK, resp)
}

// decodePromoRequest is the shared decode + validate half of both writes.
func (s *Server) decodePromoRequest(w http.ResponseWriter, r *http.Request) (domain.PromoInput, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req promoRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return domain.PromoInput{}, false
	}
	in := req.toDomain()
	if fields := domain.ValidatePromoInput(in); len(fields) > 0 {
		s.respondValidationError(w, fields)
		return domain.PromoInput{}, false
	}
	return in, true
}

// POST /admin/promos — create. 409 code_taken mirrors the categories'
// slug_taken: the unique index is case-insensitive, so "Honey10" collides
// with HONEY10.
func (s *Server) handleAdminCreatePromo(w http.ResponseWriter, r *http.Request) {
	in, ok := s.decodePromoRequest(w, r)
	if !ok {
		return
	}
	promo, err := s.store.CreatePromo(r.Context(), in)
	if err != nil {
		if errors.Is(err, domain.ErrPromoCodeTaken) {
			s.respondError(w, http.StatusConflict, "code_taken", "a promo code with this text already exists")
			return
		}
		s.log.Error("creating promo", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusCreated, toAdminPromoResponse(promo))
}

// PUT /admin/promos/{id} — whole-value update, deactivation included
// (active: false is how a code is retired; there is no DELETE).
func (s *Server) handleAdminUpdatePromo(w http.ResponseWriter, r *http.Request) {
	promoID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "promo id must be a number")
		return
	}
	in, ok := s.decodePromoRequest(w, r)
	if !ok {
		return
	}
	promo, err := s.store.UpdatePromo(r.Context(), promoID, in)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			s.respondError(w, http.StatusNotFound, "not_found", "no such promo code")
		case errors.Is(err, domain.ErrPromoCodeTaken):
			s.respondError(w, http.StatusConflict, "code_taken", "a promo code with this text already exists")
		default:
			s.log.Error("updating promo", "id", promoID, "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	s.respondJSON(w, http.StatusOK, toAdminPromoResponse(promo))
}
