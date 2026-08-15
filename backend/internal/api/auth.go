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
	// The two session lifetimes "Keep me signed in" chooses between (E8).
	// Short is the shared-computer default; long is an explicit opt-in the
	// customer makes per sign-in. Either way the token itself is rotated at
	// every login — startSession always mints a fresh one.
	sessionTTL         = 7 * 24 * time.Hour
	rememberSessionTTL = 30 * 24 * time.Hour
)

type credentialsRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// loginRequest is credentials plus the checkbox. A separate struct rather
// than a field on credentialsRequest because register also decodes that
// one under DisallowUnknownFields — and "remember" on a REGISTER body
// should be the 400 it currently is.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Remember bool   `json:"remember"`
}

type userResponse struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`

	// E7: the hive-club standing — the design's membership, which is not a
	// stored tier but a reading of the customer's order history (decision
	// #36). The SERVER derives the booleans: the client renders a badge and
	// a promise line, it does not re-implement the rule.
	Hive hiveResponse `json:"hive"`
}

type hiveResponse struct {
	PriorOrders int `json:"prior_orders"`
	// "8% less on every order after the first" — true from the second
	// order on, with the percent alongside so the UI never hardcodes it.
	Member                bool `json:"member"`
	MemberDiscountPercent int  `json:"member_discount_percent"`
	// "First order ships free" — the other perk, true until an order exists.
	FirstDeliveryFree bool `json:"first_delivery_free"`
}

func toUserResponse(u domain.User, priorOrders int) userResponse {
	hive := hiveResponse{PriorOrders: priorOrders, FirstDeliveryFree: priorOrders == 0}
	if priorOrders >= 1 {
		hive.Member = true
		hive.MemberDiscountPercent = domain.MemberDiscountPercent
	}
	return userResponse{ID: u.ID, Email: u.Email, Role: u.Role, Hive: hive}
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

// startSession creates a DB session for the user and sets the cookie. The
// TTL is a parameter since E8: the "keep me signed in" checkbox picks
// between the week and the month, and the DB row and the cookie must agree
// on whichever it is.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, userID int64, ttl time.Duration) error {
	token, err := newSessionToken()
	if err != nil {
		return err
	}
	if err := s.store.CreateSession(r.Context(), token, userID, time.Now().Add(ttl)); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(ttl.Seconds()),
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

	if err := s.startSession(w, r, user.ID, sessionTTL); err != nil {
		s.log.Error("starting session", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	// A brand-new account has zero orders by definition — no query needed
	// for its hive standing.
	s.respondJSON(w, http.StatusCreated, toUserResponse(user, 0))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req loginRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	req.Email = domain.NormalizeEmail(req.Email)

	// The limiter runs BEFORE the lookup and the bcrypt compare: it exists
	// to bound guessing, and a guess that still costs the shop a query and
	// a hash is a guess half-allowed. E8 pulls this forward from Phase 11 —
	// this is the phase with auth open on the table.
	if !s.limiter.allow(limitKey(r, req.Email)) {
		s.respondError(w, http.StatusTooManyRequests, "too_many_attempts",
			"too many attempts — try again later")
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

	// An OAuth-born account has password_hash '' and bcrypt refuses to
	// match anything against it — password login fails closed with the
	// same answer as a wrong password, no special case needed.
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.respondError(w, http.StatusUnauthorized, "invalid_credentials", domain.ErrInvalidCredentials.Error())
		return
	}

	ttl := sessionTTL
	if req.Remember {
		ttl = rememberSessionTTL
	}
	if err := s.startSession(w, r, user.ID, ttl); err != nil {
		s.log.Error("starting session", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	prior, err := s.store.PriorOrders(r.Context(), user.ID)
	if err != nil {
		s.log.Error("counting prior orders", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toUserResponse(user, prior))
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
	prior, err := s.store.PriorOrders(r.Context(), user.ID)
	if err != nil {
		s.log.Error("counting prior orders", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toUserResponse(user, prior))
}
