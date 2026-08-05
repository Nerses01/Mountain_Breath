package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The store interfaces are everything the API layer needs from the database.
// Defined HERE (at the consumer) and satisfied implicitly by *store.Store —
// in Phase 6 a fake in-memory implementation satisfies them too, letting us
// test handlers without Postgres.

type CategoryStore interface {
	ListCategories(ctx context.Context, locale domain.Locale) ([]domain.Category, error)
	CreateCategory(ctx context.Context, c *domain.Category) error
}

type ProductStore interface {
	// The locale rides inside ProductFilter, which already carries every
	// other "how should this list be shaped" option. GetProductBySlug takes
	// it as a parameter, having no filter to put it in.
	ListProducts(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int, error)
	GetProductBySlug(ctx context.Context, slug string, locale domain.Locale) (domain.Product, error)
	CreateProduct(ctx context.Context, p *domain.Product) error
	UpdateProduct(ctx context.Context, p *domain.Product) error
	UpdateVariant(ctx context.Context, variantID, priceMinor int64, stockQty int) error
	UpdateProductImage(ctx context.Context, productID int64, imageURL string) error
}

type UserStore interface {
	CreateUser(ctx context.Context, u *domain.User) error
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
}

type SessionStore interface {
	CreateSession(ctx context.Context, token string, userID int64, expiresAt time.Time) error
	GetUserBySession(ctx context.Context, token string) (domain.User, error)
	DeleteSession(ctx context.Context, token string) error
}

type CartStore interface {
	GetCart(ctx context.Context, userID int64) ([]domain.CartItem, error)
	SetCartItem(ctx context.Context, userID, variantID int64, qty int) error
	DeleteCartItem(ctx context.Context, userID, variantID int64) error
}

type OrderStore interface {
	CreateOrder(ctx context.Context, userID int64) (domain.Order, error)
	ListOrdersByUser(ctx context.Context, userID int64) ([]domain.Order, error)
	ListAllOrders(ctx context.Context) ([]domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, to string) (domain.Order, error)
}

// Store embeds the per-entity interfaces into the one the Server depends on.
type Store interface {
	CategoryStore
	ProductStore
	UserStore
	SessionStore
	CartStore
	OrderStore
}

// Server holds the dependencies of the HTTP layer. Handlers are methods on it,
// so they reach the logger and store (and later: sessions...) without globals.
type Server struct {
	log        *slog.Logger
	store      Store
	devMode    bool
	uploadsDir string
	metrics    *metrics
}

// extraCollectors lets main contribute collectors that need dependencies the
// api layer doesn't own (e.g. the pgx pool stats collector).
func NewServer(log *slog.Logger, store Store, devMode bool, uploadsDir string, extraCollectors ...prometheus.Collector) *Server {
	return &Server{
		log:        log,
		store:      store,
		devMode:    devMode,
		uploadsDir: uploadsDir,
		metrics:    newMetrics(extraCollectors...),
	}
}

func (s *Server) Routes() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(s.metricsMiddleware)
	r.Use(s.requestLogger)
	r.Use(s.recoverPanic)

	r.Get("/health", s.handleHealth)

	// Scrape endpoint for Prometheus. Reachable inside the compose network
	// and on localhost in dev — nginx does NOT proxy it, so it is never
	// exposed to the public internet.
	r.Handle("/metrics", promhttp.HandlerFor(s.metrics.registry, promhttp.HandlerOpts{}))

	// Uploaded product images. http.FileServer refuses path traversal (..)
	// on its own; filenames are server-generated anyway.
	r.Handle("/uploads/*", http.StripPrefix("/uploads/",
		http.FileServer(http.Dir(s.uploadsDir))))

	r.Route("/api/v1", func(r chi.Router) {
		// Resolve the session cookie (if any) for every API request.
		r.Use(s.withUser)
		// ...and the display language, so no handler re-derives it.
		r.Use(s.withLocale)

		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)

		r.Get("/categories", s.handleListCategories)
		r.Get("/products", s.handleListProducts)
		r.Get("/products/{slug}", s.handleGetProduct)

		// Logged-in customers: cart and checkout.
		r.Group(func(r chi.Router) {
			r.Use(s.requireUser)
			r.Get("/cart", s.handleGetCart)
			r.Put("/cart/items", s.handleSetCartItem)
			r.Delete("/cart/items/{variantID}", s.handleDeleteCartItem)
			r.Post("/orders", s.handleCreateOrder)
			r.Get("/orders", s.handleListMyOrders)
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(s.requireAdmin)
			r.Post("/categories", s.handleCreateCategory)
			r.Get("/orders", s.handleAdminListOrders)
			r.Patch("/orders/{id}/status", s.handleUpdateOrderStatus)
			r.Get("/products", s.handleAdminListProducts)
			r.Post("/products", s.handleCreateProduct)
			r.Put("/products/{id}", s.handleUpdateProduct)
			r.Post("/products/{id}/image", s.handleUploadProductImage)
			r.Patch("/variants/{id}", s.handleUpdateVariant)
		})
	})

	if s.devMode {
		// Deliberately slow endpoint for observing graceful shutdown
		// and client cancellation. Never registered in prod.
		r.Get("/debug/slow", s.handleSlow)
	}

	return r
}
