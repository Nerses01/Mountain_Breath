package api

import (
	"context"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The store interfaces are everything the API layer needs from the database.
// Defined HERE (at the consumer) and satisfied implicitly by *store.Store —
// in Phase 6 a fake in-memory implementation satisfies them too, letting us
// test handlers without Postgres.

type CategoryStore interface {
	ListCategories(ctx context.Context) ([]domain.Category, error)
	CreateCategory(ctx context.Context, c *domain.Category) error
}

type ProductStore interface {
	ListProducts(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int, error)
	GetProductBySlug(ctx context.Context, slug string) (domain.Product, error)
}

// Store embeds the per-entity interfaces into the one the Server depends on.
type Store interface {
	CategoryStore
	ProductStore
}

// Server holds the dependencies of the HTTP layer. Handlers are methods on it,
// so they reach the logger and store (and later: sessions...) without globals.
type Server struct {
	log     *slog.Logger
	store   Store
	devMode bool
}

func NewServer(log *slog.Logger, store Store, devMode bool) *Server {
	return &Server{log: log, store: store, devMode: devMode}
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(s.requestLogger)
	r.Use(s.recoverPanic)

	r.Get("/health", s.handleHealth)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/categories", s.handleListCategories)
		r.Post("/categories", s.handleCreateCategory) // TODO Phase 4: admin-only

		r.Get("/products", s.handleListProducts)
		r.Get("/products/{slug}", s.handleGetProduct)
	})

	if s.devMode {
		// Deliberately slow endpoint for observing graceful shutdown
		// and client cancellation. Never registered in prod.
		r.Get("/debug/slow", s.handleSlow)
	}

	return r
}
