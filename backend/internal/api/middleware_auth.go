package api

import (
	"context"
	"net/http"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// Unexported key type: no other package can collide with (or read) this
// context entry by accident.
type ctxKey struct{}

// withUser resolves the session cookie to a user and attaches it to the
// request context. It never rejects — endpoints decide what anonymity means.
func (s *Server) withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
			if user, err := s.store.GetUserBySession(r.Context(), c.Value); err == nil {
				r = r.WithContext(context.WithValue(r.Context(), ctxKey{}, user))
			}
		}
		next.ServeHTTP(w, r)
	})
}

func userFrom(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(ctxKey{}).(domain.User)
	return u, ok
}

// requireAdmin gates a subtree: 401 for anonymous, 403 for non-admins.
// (401 = "who are you?", 403 = "I know you, and no.")
func (s *Server) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := userFrom(r.Context())
		if !ok {
			s.respondError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		if !user.IsAdmin() {
			s.respondError(w, http.StatusForbidden, "forbidden", "admin access required")
			return
		}
		next.ServeHTTP(w, r)
	})
}
