package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The account page's address book (E8) — the "several named addresses" E6
// deferred. The checkout keeps its inline form and its default-address
// prefill; this CRUD is how the book behind that prefill is managed.

type addressEntryPayload struct {
	ID        int64  `json:"id,omitempty"`
	Label     string `json:"label"`
	IsDefault bool   `json:"is_default"`
	addressPayload
}

func toAddressEntryPayload(e domain.AddressEntry) addressEntryPayload {
	return addressEntryPayload{
		ID: e.ID, Label: e.Label, IsDefault: e.IsDefault,
		addressPayload: toAddressPayload(e.Address),
	}
}

func (p addressEntryPayload) toDomain() domain.AddressEntry {
	return domain.AddressEntry{
		ID: p.ID, Label: p.Label, IsDefault: p.IsDefault,
		Address: p.addressPayload.toDomain(),
	}
}

func (s *Server) handleListAddresses(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	entries, err := s.store.ListAddresses(r.Context(), user.ID)
	if err != nil {
		s.log.Error("listing addresses", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp := make([]addressEntryPayload, 0, len(entries))
	for _, e := range entries {
		resp = append(resp, toAddressEntryPayload(e))
	}
	s.respondJSON(w, http.StatusOK, resp)
}

// decodeAddressEntry shares the validation path with the checkout: the same
// domain.ValidateAddress, the same field keys, so the account form and the
// checkout form speak one error language.
func (s *Server) decodeAddressEntry(w http.ResponseWriter, r *http.Request) (domain.AddressEntry, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req addressEntryPayload
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return domain.AddressEntry{}, false
	}
	entry := req.toDomain()
	fields := domain.ValidateAddress(entry.Address)
	if len(entry.Label) > 60 {
		fields["label"] = domain.ValidationTooLong
	}
	if len(fields) > 0 {
		s.respondValidationError(w, fields)
		return domain.AddressEntry{}, false
	}
	return entry, true
}

func (s *Server) handleCreateAddress(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	entry, ok := s.decodeAddressEntry(w, r)
	if !ok {
		return
	}

	created, err := s.store.CreateAddress(r.Context(), user.ID, entry)
	if err != nil {
		s.log.Error("creating address", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusCreated, toAddressEntryPayload(created))
}

func (s *Server) handleUpdateAddress(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	addressID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "address id must be a number")
		return
	}
	entry, ok := s.decodeAddressEntry(w, r)
	if !ok {
		return
	}
	entry.ID = addressID

	if err := s.store.UpdateAddress(r.Context(), user.ID, entry); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Someone else's address behaves exactly like no address — the
			// WHERE user_id clause is the authorization, and 404 is its
			// honest surface (the order-privacy rule again).
			s.respondError(w, http.StatusNotFound, "not_found", "no such address")
			return
		}
		s.log.Error("updating address", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toAddressEntryPayload(entry))
}

func (s *Server) handleDeleteAddress(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	addressID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "address id must be a number")
		return
	}

	if err := s.store.DeleteAddress(r.Context(), user.ID, addressID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such address")
			return
		}
		s.log.Error("deleting address", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
