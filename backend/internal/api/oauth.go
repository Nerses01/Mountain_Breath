package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// "Continue with Google" (decision #5) — the OAuth 2.0 authorization-code
// flow, hand-rolled. It is three redirects and one back-channel call, and
// naming them is most of understanding OAuth:
//
//  1. WE redirect the browser to Google, carrying a random `state` we also
//     set as a short-lived cookie. state is our CSRF proof: only a flow WE
//     started can finish.
//  2. Google authenticates the person however it likes and redirects them
//     back to our callback with a one-time `code`.
//  3. THE SERVER exchanges code for tokens — a POST from our process to
//     Google's, carrying the client secret the browser must never see.
//     This server-to-server leg is what "authorization code" means: the
//     browser only ever transports an opaque one-time voucher.
//  4. With the access token we ask the userinfo endpoint who the person is
//     (subject id + verified email). Because the token arrived over OUR
//     TLS connection to Google, the answer is trustworthy without any JWT
//     signature verification — the id_token + JWKS dance exists for
//     parties who receive tokens they did not fetch themselves.

type googleOAuth struct {
	clientID     string
	clientSecret string
	publicURL    string

	// Endpoint overrides for tests, which stand up a fake Google with
	// httptest. Empty = the real service.
	authURL     string
	tokenURL    string
	userinfoURL string
}

func (g googleOAuth) enabled() bool { return g.clientID != "" && g.clientSecret != "" }

func (g googleOAuth) authEndpoint() string {
	if g.authURL != "" {
		return g.authURL
	}
	return "https://accounts.google.com/o/oauth2/v2/auth"
}

func (g googleOAuth) tokenEndpoint() string {
	if g.tokenURL != "" {
		return g.tokenURL
	}
	return "https://oauth2.googleapis.com/token"
}

func (g googleOAuth) userinfoEndpoint() string {
	if g.userinfoURL != "" {
		return g.userinfoURL
	}
	return "https://openidconnect.googleapis.com/v1/userinfo"
}

// redirectURI goes through the PUBLIC origin (Vite in dev, nginx in prod),
// not the API's own address: it must match what is registered in the Google
// console, and it must be an address the visitor's browser can reach.
func (g googleOAuth) redirectURI() string {
	return g.publicURL + "/api/v1/auth/oauth/google/callback"
}

const oauthStateCookie = "mb_oauth_state"

var oauthHTTPClient = &http.Client{Timeout: 10 * time.Second}

// GET /auth/oauth/google — leg 1: send the browser to Google.
func (s *Server) handleGoogleStart(w http.ResponseWriter, r *http.Request) {
	if !s.google.enabled() {
		// Unconfigured = the feature does not exist. The login page knows
		// (its button explains itself); a hand-typed URL gets a plain 404.
		s.respondError(w, http.StatusNotFound, "not_found", "google sign-in is not configured")
		return
	}

	state, err := newSessionToken()
	if err != nil {
		s.log.Error("generating oauth state", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	// The locale rides in the cookie beside the state so the callback can
	// land the visitor back in their own language — the state parameter
	// itself stays pure randomness.
	locale := localeFromContext(r.Context())
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookie,
		Value:    state + "|" + string(locale),
		Path:     "/",
		MaxAge:   600, // the flow takes seconds; ten minutes is generous
		HttpOnly: true,
		Secure:   !s.devMode,
		SameSite: http.SameSiteLaxMode,
	})

	q := url.Values{
		"client_id":     {s.google.clientID},
		"redirect_uri":  {s.google.redirectURI()},
		"response_type": {"code"},
		// openid+email is the whole ask: we need an identity and a verified
		// address, not the contact list. Minimal scopes are also what keeps
		// the consent screen unscary.
		"scope": {"openid email"},
		"state": {state},
	}
	http.Redirect(w, r, s.google.authEndpoint()+"?"+q.Encode(), http.StatusFound)
}

// GET /auth/oauth/google/callback — legs 2–4, then a session.
//
// Every failure here redirects to the login page with ?oauth_error=1 rather
// than rendering JSON: the visitor arrives by browser navigation, and a
// wall of {"error":…} would be the shop shrugging at them.
func (s *Server) handleGoogleCallback(w http.ResponseWriter, r *http.Request) {
	if !s.google.enabled() {
		s.respondError(w, http.StatusNotFound, "not_found", "google sign-in is not configured")
		return
	}

	// The state cookie is single-use whatever happens next.
	locale := domain.DefaultLocale
	var cookieState string
	if c, err := r.Cookie(oauthStateCookie); err == nil {
		if state, loc, ok := strings.Cut(c.Value, "|"); ok {
			cookieState = state
			if parsed, ok := domain.ParseLocale(loc); ok {
				locale = parsed
			}
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: oauthStateCookie, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: !s.devMode, SameSite: http.SameSiteLaxMode,
	})

	fail := func(why string, err error) {
		s.log.Warn("google sign-in failed", "why", why, "error", err)
		http.Redirect(w, r,
			s.publicURL+localePathPrefix(locale)+"/login?oauth_error=1",
			http.StatusFound)
	}

	if e := r.URL.Query().Get("error"); e != "" {
		// The person clicked "cancel" on Google's screen — not an attack,
		// still not a session.
		fail("provider returned "+e, nil)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" || cookieState == "" || r.URL.Query().Get("state") != cookieState {
		// A missing or wrong state means this callback was not the end of a
		// flow WE started — the CSRF case the state parameter exists for.
		fail("state mismatch", nil)
		return
	}

	subject, email, err := s.google.exchange(r.Context(), code)
	if err != nil {
		fail("code exchange", err)
		return
	}

	user, err := s.store.FindOrCreateOAuthUser(r.Context(), "google", subject, email)
	if err != nil {
		fail("resolving account", err)
		return
	}
	if err := s.startSession(w, r, user.ID, sessionTTL); err != nil {
		fail("starting session", err)
		return
	}
	http.Redirect(w, r, s.publicURL+localePathPrefix(locale)+"/", http.StatusFound)
}

// exchange is legs 3 and 4: code → token → identity.
func (g googleOAuth) exchange(ctx context.Context, code string) (subject, email string, err error) {
	form := url.Values{
		"code":          {code},
		"client_id":     {g.clientID},
		"client_secret": {g.clientSecret},
		"redirect_uri":  {g.redirectURI()},
		"grant_type":    {"authorization_code"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.tokenEndpoint(), strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := oauthHTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("token request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("token endpoint answered %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return "", "", fmt.Errorf("decoding token response: %w", err)
	}
	if tok.AccessToken == "" {
		return "", "", fmt.Errorf("token response carried no access token")
	}

	uiReq, err := http.NewRequestWithContext(ctx, http.MethodGet, g.userinfoEndpoint(), nil)
	if err != nil {
		return "", "", err
	}
	uiReq.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	uiResp, err := oauthHTTPClient.Do(uiReq)
	if err != nil {
		return "", "", fmt.Errorf("userinfo request: %w", err)
	}
	defer func() { _ = uiResp.Body.Close() }()
	if uiResp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("userinfo endpoint answered %d", uiResp.StatusCode)
	}
	var info struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := json.NewDecoder(uiResp.Body).Decode(&info); err != nil {
		return "", "", fmt.Errorf("decoding userinfo: %w", err)
	}
	if info.Sub == "" || info.Email == "" {
		return "", "", fmt.Errorf("userinfo missing sub or email")
	}
	// The verified check is what makes email-linking safe (see
	// FindOrCreateOAuthUser): an unverified address is a claim anyone can
	// type at a provider, and linking on it would hand over the account
	// that really owns the address.
	if !info.EmailVerified {
		return "", "", fmt.Errorf("google email is unverified")
	}
	return info.Sub, domain.NormalizeEmail(info.Email), nil
}
