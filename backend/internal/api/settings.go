package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// A5 (canvas 10): the settings screen's write paths — profile, password,
// and the notifications panel's two REAL channels (decision log #87).

type profileRequest struct {
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

// PATCH /account/profile — the two fields the settings form edits. Email
// is deliberately not here: changing the account's identity is a
// different, verification-shaped problem, not a form field.
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req profileRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	if fields := domain.ValidateProfile(req.FullName, req.Phone); len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	if err := s.store.UpdateProfile(r.Context(), user.ID, req.FullName, req.Phone); err != nil {
		s.log.Error("updating profile", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// Answer with the fresh user, so the client's ['me'] cache can be set
	// from the response instead of refetched.
	user.FullName, user.Phone = req.FullName, req.Phone
	prior, err := s.store.PriorOrders(r.Context(), user.ID)
	if err != nil {
		s.log.Error("counting prior orders", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toUserResponse(user, prior))
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// POST /account/password — the signed-in rotation (A5). The contract the
// tests pin: the CURRENT password is verified first (a stolen unlocked
// laptop must not be enough to lock the owner out), the new one meets the
// reset flow's same length rule, and every OTHER session is revoked — the
// person changing the password stays signed in, anyone else holding a
// cookie does not.
func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req changePasswordRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	if len(req.NewPassword) < domain.PasswordMinLength {
		s.respondValidationError(w, map[string]string{"new_password": domain.ValidationPasswordTooShort})
		return
	}

	// An OAuth-born account has password_hash "" and bcrypt fails closed —
	// the same no-special-case honesty as login. Its owner sets a first
	// password through the reset flow, which verifies the inbox instead.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)); err != nil {
		// A field error, not a 401: the SESSION is fine, one input is wrong.
		s.respondValidationError(w, map[string]string{"current_password": domain.ValidationIncorrectPassword})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("hashing password", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// The caller's own session survives; it is named by the cookie that
	// authenticated this very request.
	keep := ""
	if c, err := r.Cookie(sessionCookieName); err == nil {
		keep = c.Value
	}
	if err := s.store.ChangePassword(r.Context(), user.ID, string(hash), keep); err != nil {
		s.log.Error("changing password", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type notificationsResponse struct {
	OrderUpdates bool `json:"order_updates"`
	// none | pending | subscribed — the harvest-notes toggle's real state,
	// read from the newsletter table by the account's email.
	Newsletter string `json:"newsletter"`
}

// GET /account/notifications — what the panel's two real toggles show.
func (s *Server) handleGetNotifications(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	status, err := s.store.NewsletterStatusByEmail(r.Context(), user.Email)
	if err != nil {
		s.log.Error("reading newsletter status", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, notificationsResponse{
		OrderUpdates: user.NotifyOrderUpdates,
		Newsletter:   status,
	})
}

type notificationsRequest struct {
	OrderUpdates bool `json:"order_updates"`
}

// PATCH /account/notifications — the order-updates toggle. The harvest
// toggle is NOT here: on goes through the newsletter's own double-opt-in
// (POST /newsletter/subscribe — consent stays verified), off through
// DELETE /account/newsletter below.
func (s *Server) handleSetNotifications(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req notificationsRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	if err := s.store.SetNotifyOrderUpdates(r.Context(), user.ID, req.OrderUpdates); err != nil {
		s.log.Error("setting notification preference", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DELETE /account/newsletter — the harvest toggle's OFF. No token needed:
// being signed in to the account IS proof of owning the email; the emailed
// token remains the anonymous path.
func (s *Server) handleAccountNewsletterUnsubscribe(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())

	if err := s.store.UnsubscribeNewsletterByEmail(r.Context(), user.Email); err != nil && !errors.Is(err, domain.ErrNotFound) {
		s.log.Error("unsubscribing by account", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
