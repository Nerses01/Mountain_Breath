package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/mail"
)

// The newsletter (E9): subscribe with double opt-in, confirm, unsubscribe.
//
// ONE DELIBERATE DEPARTURE from the plan, worth its own paragraph: the plan
// wrote `GET /newsletter/confirm?token=`, the classic emailed-link shape —
// and that shape has a documented failure mode. Corporate mail scanners
// (Outlook Safe Links and friends) PREFETCH every GET link in an incoming
// mail to check it for malware, which means a mutating GET gets "clicked"
// by a robot before the human ever opens the message — auto-confirming
// subscriptions the person never approved, which is precisely what double
// opt-in exists to prevent. So the emailed link lands on a FRONTEND page,
// and the page POSTs the token here. Scanners follow GETs; they do not
// submit forms.

type subscribeRequest struct {
	Email string `json:"email"`
}

// POST /newsletter/subscribe — the footer form. Answers 204 whatever the
// address's history (never seen, pending, already subscribed): the form
// must not be an oracle for "is this address on the list". Rate limited,
// because an endpoint that mails arbitrary addresses on request is
// otherwise a spam cannon with the shop's name on it.
func (s *Server) handleNewsletterSubscribe(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req subscribeRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}
	email := domain.NormalizeEmail(req.Email)
	if !domain.ValidEmail(email) {
		s.respondValidationError(w, map[string]string{"email": domain.ValidationEmailFormat})
		return
	}
	if !s.limiter.allow(limitKey(r, email)) {
		s.respondError(w, http.StatusTooManyRequests, "too_many_attempts",
			"too many attempts — try again later")
		return
	}

	token, err := newSessionToken()
	if err != nil {
		s.log.Error("generating newsletter token", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	needsConfirmation, err := s.store.SubscribeNewsletter(r.Context(), email, token)
	if err != nil {
		s.log.Error("subscribing to newsletter", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// A live subscriber gets no mail — re-mailing "please confirm" on every
	// footer submit would be the shop spamming its own most loyal readers.
	if needsConfirmation {
		locale := localeFromContext(r.Context())
		link := s.publicURL + localePathPrefix(locale) + "/newsletter/confirm/" + token
		if err := s.mailer.Send(r.Context(), mail.NewsletterConfirmMessage(locale, email, link)); err != nil {
			s.log.Error("sending newsletter confirmation", "error", err)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

type newsletterTokenRequest struct {
	Token string `json:"token"`
}

func (s *Server) decodeNewsletterToken(w http.ResponseWriter, r *http.Request) (string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req newsletterTokenRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return "", false
	}
	if req.Token == "" {
		s.respondError(w, http.StatusBadRequest, "invalid_token", "this link is not valid")
		return "", false
	}
	return req.Token, true
}

// POST /newsletter/confirm — the emailed link's page posts the token back.
// Idempotent (people re-click links); only an invented token fails.
func (s *Server) handleNewsletterConfirm(w http.ResponseWriter, r *http.Request) {
	token, ok := s.decodeNewsletterToken(w, r)
	if !ok {
		return
	}
	if err := s.store.ConfirmNewsletter(r.Context(), token); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusBadRequest, "invalid_token", "this link is not valid")
			return
		}
		s.log.Error("confirming newsletter", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /newsletter/unsubscribe — the same capability, pointed the other way.
func (s *Server) handleNewsletterUnsubscribe(w http.ResponseWriter, r *http.Request) {
	token, ok := s.decodeNewsletterToken(w, r)
	if !ok {
		return
	}
	if err := s.store.UnsubscribeNewsletter(r.Context(), token); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusBadRequest, "invalid_token", "this link is not valid")
			return
		}
		s.log.Error("unsubscribing from newsletter", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
