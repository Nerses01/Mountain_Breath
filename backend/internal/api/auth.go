package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

const (
	sessionCookieName = "mb_session"
	sessionTTL        = 7 * 24 * time.Hour
)

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func toUserResponse(u domain.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Role: u.Role}
}

// newSessionToken returns 32 cryptographically random bytes as hex.
// math/rand would be a security hole here — it's predictable.
func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (s *Server) decodeCredentials(w http.ResponseWriter, r *http.Request) (credentialsRequest, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req credentialsRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return credentialsRequest{}, false
	}
	req.Email = domain.NormalizeEmail(req.Email)
	return req, true
}

// startSession creates a DB session for the user and sets the cookie.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64) error {
	token, err := newSessionToken()
	if err != nil {
		return err
	}
	if err := s.store.CreateSession(r.Context(), token, userID, time.Now().Add(sessionTTL)); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,               // JavaScript cannot read it → XSS can't steal it
		Secure:   !s.devMode,         // HTTPS-only in prod; localhost has no TLS
		SameSite: http.SameSiteLaxMode, // not sent on cross-site POSTs → CSRF baseline
	})
	return nil
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	req, ok := s.decodeCredentials(w, r)
	if !ok {
		return
	}

	if fields := domain.ValidateRegistration(req.Email, req.Password); len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("hashing password", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	user := domain.User{Email: req.Email, PasswordHash: string(hash), Role: domain.RoleCustomer}
	if err := s.store.CreateUser(r.Context(), &user); err != nil {
		if errors.Is(err, domain.ErrEmailTaken) {
			s.respondError(w, http.StatusConflict, "email_taken", "this email is already registered")
			return
		}
		s.log.Error("creating user", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	if err := s.startSession(w, r, user.ID); err != nil {
		s.log.Error("starting session", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusCreated, toUserResponse(user))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	req, ok := s.decodeCredentials(w, r)
	if !ok {
		return
	}

	user, err := s.store.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// Same response as a wrong password — do not reveal that the
			// email is unknown (user enumeration).
			s.respondError(w, http.StatusUnauthorized, "invalid_credentials", domain.ErrInvalidCredentials.Error())
			return
		}
		s.log.Error("looking up user", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.respondError(w, http.StatusUnauthorized, "invalid_credentials", domain.ErrInvalidCredentials.Error())
		return
	}

	if err := s.startSession(w, r, user.ID); err != nil {
		s.log.Error("starting session", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toUserResponse(user))
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
		if err := s.store.DeleteSession(r.Context(), c.Value); err != nil {
			s.log.Error("deleting session", "error", err)
		}
	}
	// Overwrite the cookie with an immediately-expiring one.
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   !s.devMode,
		SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFrom(r.Context())
	if !ok {
		s.respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	s.respondJSON(w, http.StatusOK, toUserResponse(user))
}
