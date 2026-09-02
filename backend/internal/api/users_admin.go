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

// F2 (decision #96): user administration — promotion stops being a psql
// incantation (the Era I leftover). Note what is NOT here: no user
// creation (registration owns that), no deletion (that is the privacy
// item's job, with its own rules), no email/password edits (those belong
// to the account's owner alone). The admin manages ROLES, nothing else.

type adminUserResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	// All orders ever, cancelled included — "how much history does this
	// account have", not revenue.
	Orders int `json:"orders"`
}

// GET /admin/users — everyone, newest first.
func (s *Server) handleAdminListUsers(w http.ResponseWriter, r *http.Request) {
	users, counts, err := s.store.ListUsers(r.Context())
	if err != nil {
		s.log.Error("listing users", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	resp := make([]adminUserResponse, 0, len(users))
	for i, u := range users {
		resp = append(resp, adminUserResponse{
			ID: u.ID, Email: u.Email, FullName: u.FullName,
			Role: u.Role, CreatedAt: u.CreatedAt, Orders: counts[i],
		})
	}
	s.respondJSON(w, http.StatusOK, resp)
}

type updateRoleRequest struct {
	Role string `json:"role"`
}

// PATCH /admin/users/{id}/role — promote or demote. The "at least one
// admin" invariant is the STORE's (counted under locks); this handler
// translates its refusal into 409 last_admin, which covers the plan's
// "the last admin cannot demote themself" as a special case of the
// general rule — self or not, the count may not reach zero.
func (s *Server) handleUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	userID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "user id must be a number")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req updateRoleRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	if !domain.ValidRole(req.Role) {
		s.respondValidationError(w, map[string]string{"role": "unknown role"})
		return
	}

	user, err := s.store.UpdateUserRole(r.Context(), userID, req.Role)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			s.respondError(w, http.StatusNotFound, "not_found", "no such user")
		case errors.Is(err, domain.ErrLastAdmin):
			s.respondError(w, http.StatusConflict, "last_admin",
				"the shop must keep at least one admin — promote someone else first")
		default:
			s.log.Error("updating user role", "id", userID, "error", err)
			s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		}
		return
	}
	s.respondJSON(w, http.StatusOK, adminUserResponse{
		ID: user.ID, Email: user.Email, FullName: user.FullName,
		Role: user.Role, CreatedAt: user.CreatedAt,
	})
}
