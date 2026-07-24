package api

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server holds the dependencies of the HTTP layer. Handlers are methods on it,
// so they reach the logger (and later: the store, sessions...) without globals.
type Server struct {
	log     *slog.Logger
	devMode bool
}

func NewServer(log *slog.Logger, devMode bool) *Server {
	return &Server{log: log, devMode: devMode}
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(s.requestLogger)
	r.Use(s.recoverPanic)

	r.Get("/health", s.handleHealth)

	if s.devMode {
		// Deliberately slow endpoint for observing graceful shutdown
		// and client cancellation. Never registered in prod.
		r.Get("/debug/slow", s.handleSlow)
	}

	return r
}
