package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/mail"
)

// resetTokenTTL is deliberately short: the link is a temporary password
// sitting in an inbox, and inboxes get compromised LATER — the fuse bounds
// how much later still matters.
const resetTokenTTL = time.Hour

// localePathPrefix mirrors the frontend's route structure (E1.5): bare
// paths are English, /hy and /ru prefix the others. Emailed links must
// land the reader in the language they asked from.
func localePathPrefix(l domain.Locale) string {
	if l == domain.DefaultLocale {
		return ""
	}
	return "/" + string(l)
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

// POST /auth/forgot-password — request a reset link.
//
// The response is 204 WHETHER OR NOT THE EMAIL EXISTS. Login already
// answers identically for unknown email and wrong password; a reset
// endpoint that said "sent!" for members and "no such account" for
// strangers would undo that — it would be a free membership oracle for
// every address on earth. The cost is honest: a person who typoes their
// address waits for a mail that never comes. The rate limiter above the
// lookup also keeps this endpoint from being a spam cannon pointed at
// arbitrary inboxes.
func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req forgotPasswordRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	email := domain.NormalizeEmail(req.Email)
	if email == "" {
		s.respondValidationError(w, map[string]string{"email": domain.ValidationRequired})
		return
	}

	if !s.limiter.allow(limitKey(r, email)) {
		s.respondError(w, http.StatusTooManyRequests, "too_many_attempts",
			"too many attempts — try again later")
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			w.WriteHeader(http.StatusNoContent) // indistinguishable from success
			return
		}
		s.log.Error("looking up user for reset", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	token, err := newSessionToken() // same 256-bit randomness as sessions
	if err != nil {
		s.log.Error("generating reset token", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	if err := s.store.CreatePasswordReset(r.Context(), user.ID, token, time.Now().Add(resetTokenTTL)); err != nil {
		s.log.Error("storing reset token", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	locale := localeFromContext(r.Context())
	link := s.publicURL + localePathPrefix(locale) + "/reset-password/" + token
	if err := s.mailer.Send(r.Context(), mail.ResetMessage(locale, user.Email, link)); err != nil {
		// Logged, not surfaced: a mail hiccup must not tell the CALLER
		// anything the anti-enumeration answer was hiding.
		s.log.Error("sending reset mail", "error", err)
	}
	w.WriteHeader(http.StatusNoContent)
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// POST /auth/reset-password — spend the emailed token, set the new password.
func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req resetPasswordRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	if len(req.Password) < domain.PasswordMinLength {
		s.respondValidationError(w, map[string]string{"password": domain.ValidationPasswordTooShort})
		return
	}
	if req.Token == "" {
		s.respondError(w, http.StatusBadRequest, "invalid_token", "this reset link is not valid")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("hashing password", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	if err := s.store.ConsumePasswordReset(r.Context(), req.Token, string(hash)); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Unknown, already spent and expired all read the same — one
			// message, one code, nothing for a token-guesser to calibrate
			// against. 400, not 404: the resource here is the REQUEST.
			s.respondError(w, http.StatusBadRequest, "invalid_token", "this reset link is not valid")
			return
		}
		s.log.Error("consuming reset token", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// Every session was revoked with the password; the customer signs in
	// fresh. Not auto-signing them in keeps this endpoint's power narrow:
	// it changes a password, it does not mint sessions.
	w.WriteHeader(http.StatusNoContent)
}
