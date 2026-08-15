package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/mail"
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
	// The locale and currency ride inside ProductFilter, which already
	// carries every other "how should this list be shaped" option. The
	// single-product reads have no filter to put them in, so they take a
	// domain.View — one value rather than two loose strings that could be
	// passed in the wrong order.
	ListProducts(ctx context.Context, f domain.ProductFilter) ([]domain.Product, int, error)
	// CatalogFacets takes the same filter as the listing — the sidebar
	// counts describe the same query the grid runs, minus paging.
	CatalogFacets(ctx context.Context, f domain.ProductFilter) (domain.CatalogFacets, error)
	GetProductBySlug(ctx context.Context, slug string, view domain.View) (domain.Product, error)
	// ListRelated is "Often taken together": the admin's curated list, or a
	// shared-benefit-then-popularity ranking when nothing is curated.
	ListRelated(ctx context.Context, slug string, view domain.View) ([]domain.Product, error)
	// ListCuratedRelated is the same question without the fallback — what
	// the admin actually chose, which is the only version a picker can
	// safely pre-fill from.
	ListCuratedRelated(ctx context.Context, slug string, view domain.View) ([]domain.Product, error)
	CreateProduct(ctx context.Context, p *domain.Product) error
	UpdateProduct(ctx context.Context, p *domain.Product) error
	UpdateVariant(ctx context.Context, variantID int64, prices domain.Money, stockQty int) error
	UpdateProductImage(ctx context.Context, productID int64, imageURL string) error

	// E3 editorial writes. Collections are replaced wholesale — see the note
	// at the top of store/products_admin_detail.go for why.
	AddProductImage(ctx context.Context, productID int64, url string, alts map[domain.Locale]string) (domain.ProductImage, error)
	SaveProductImages(ctx context.Context, productID int64, images []domain.ProductImage, alts map[int64]map[domain.Locale]string) error
	DeleteProductImage(ctx context.Context, productID, imageID int64) error
	SaveProductEditorial(ctx context.Context, productID int64, byLocale map[domain.Locale]domain.ProductEditorial) error
	SaveProductRelated(ctx context.Context, productID int64, relatedIDs []int64) error
}

// ReviewStore is E4's slice. CanReview is separate from ListReviews because
// it answers a question about the VIEWER, not about the data — and the
// product detail handler needs it without loading a single review.
type ReviewStore interface {
	ListReviews(ctx context.Context, f domain.ReviewFilter) ([]domain.Review, int, error)
	CreateReview(ctx context.Context, r *domain.Review, productSlug string) error
	UpdateReviewStatus(ctx context.Context, reviewID int64, status string) (domain.Review, error)
	CanReview(ctx context.Context, userID int64, productSlug string) (bool, error)
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
	// The view is a parameter here for the same reason it is on
	// GetProductBySlug: the cart shows product NAMES and PRICES, and a
	// basket in the wrong language — or the wrong currency — is as wrong as
	// a catalog in the wrong language.
	GetCart(ctx context.Context, userID int64, view domain.View) ([]domain.CartItem, error)
	SetCartItem(ctx context.Context, userID, variantID int64, qty int) error
	DeleteCartItem(ctx context.Context, userID, variantID int64) error
}

type OrderStore interface {
	// The currency is what the customer is CHARGED in, so it is decided at
	// the edge (withCurrency) and stamped on the order — not read back off
	// the cart, which has no single currency of its own. The CheckoutInput
	// carries the customer's CHOICES and no money; see domain.CheckoutInput.
	CreateOrder(ctx context.Context, userID int64, currency domain.Currency, in domain.CheckoutInput) (domain.Order, error)
	// GetOrder is unscoped; the handler decides who may see it (owner or
	// admin) — the store answers "what is order 12", not "may you look".
	GetOrder(ctx context.Context, orderID int64) (domain.Order, error)
	ListOrdersByUser(ctx context.Context, userID int64) ([]domain.Order, error)
	ListAllOrders(ctx context.Context) ([]domain.Order, error)
	UpdateOrderStatus(ctx context.Context, orderID int64, to string) (domain.Order, error)
}

// CheckoutStore is E6's slice: the pieces a checkout screen needs before an
// order exists — a shipping quote for the summary card and the saved
// address for pre-filling the form.
type CheckoutStore interface {
	ShippingRates(ctx context.Context) (map[domain.Currency]domain.ShippingRate, error)
	DefaultAddress(ctx context.Context, userID int64) (domain.Address, error)
}

// PromoStore is E7's slice: codes, the cart's applied code, and the two
// hive-club facts. PriorOrders lives here rather than on OrderStore because
// its callers are the pricing path (preview, /auth/me), not order CRUD.
type PromoStore interface {
	// PromoForUser resolves a typed code WITH the asking user's usage state
	// filled in, so domain.Promo.Issue can judge it without a second query.
	PromoForUser(ctx context.Context, code string, userID int64) (domain.Promo, error)
	// nil, nil when the cart carries no code — absence is the normal state,
	// not an error.
	CartPromoForUser(ctx context.Context, userID int64) (*domain.Promo, error)
	SetCartPromo(ctx context.Context, userID, codeID int64) error
	ClearCartPromo(ctx context.Context, userID int64) error
	// Non-cancelled orders — the one number both hive-club perks derive from.
	PriorOrders(ctx context.Context, userID int64) (int, error)
	// The free-shipping banner's suggestion; nil when no product closes the gap.
	UpsellForGap(ctx context.Context, view domain.View, gapMinor int64) (*domain.Upsell, error)
}

// AccountStore is E8's slice: the hearts, the reset tokens, the address
// book and the OAuth identities — everything the account grows this phase.
type AccountStore interface {
	// The wishlist answers with full product CARDS; the heart's on/off
	// state everywhere derives from this one list client-side.
	ListWishlist(ctx context.Context, userID int64, view domain.View) ([]domain.Product, error)
	AddWishlistItem(ctx context.Context, userID, productID int64) error
	RemoveWishlistItem(ctx context.Context, userID, productID int64) error
	// One transaction: the cart line and the wishlist row must never both
	// exist, nor neither.
	SaveForLater(ctx context.Context, userID, variantID int64) error

	// The sessions pattern re-armed: raw token to the inbox, SHA-256 to the
	// table, single use enforced under a row lock.
	CreatePasswordReset(ctx context.Context, userID int64, token string, expiresAt time.Time) error
	ConsumePasswordReset(ctx context.Context, token, newPasswordHash string) error

	ListAddresses(ctx context.Context, userID int64) ([]domain.AddressEntry, error)
	CreateAddress(ctx context.Context, userID int64, e domain.AddressEntry) (domain.AddressEntry, error)
	UpdateAddress(ctx context.Context, userID int64, e domain.AddressEntry) error
	DeleteAddress(ctx context.Context, userID, addressID int64) error

	// Provider identity → shop account: known subject wins, verified email
	// links, otherwise a fresh passwordless customer.
	FindOrCreateOAuthUser(ctx context.Context, provider, subject, email string) (domain.User, error)
}

// NewsletterStore is E9's slice: double opt-in, and the token as the
// subscriber's permanent capability (confirm now, unsubscribe forever).
type NewsletterStore interface {
	// needsConfirmation is false for an already-live subscriber — the
	// handler then sends nothing, but still answers 204.
	SubscribeNewsletter(ctx context.Context, email, token string) (needsConfirmation bool, err error)
	ConfirmNewsletter(ctx context.Context, token string) error
	UnsubscribeNewsletter(ctx context.Context, token string) error
}

// Store embeds the per-entity interfaces into the one the Server depends on.
type Store interface {
	CategoryStore
	ProductStore
	ReviewStore
	UserStore
	SessionStore
	CartStore
	OrderStore
	CheckoutStore
	PromoStore
	AccountStore
	NewsletterStore
}

// Options are the E8 dependencies main wires in: how to send mail, the
// origin the BROWSER uses (for emailed links and OAuth redirects — the
// API's own address would be wrong in both), and the Google client. A
// struct rather than more positional parameters, because a zero Options is
// a meaningful configuration (log-sink mail, OAuth disabled) that every
// existing test gets for free.
type Options struct {
	Mailer             mail.Mailer
	PublicURL          string
	GoogleClientID     string
	GoogleClientSecret string
	// Endpoint overrides so handler tests can stand up a FAKE Google with
	// httptest and drive the whole callback path. Empty = the real service.
	GoogleAuthURL     string
	GoogleTokenURL    string
	GoogleUserinfoURL string
}

// Server holds the dependencies of the HTTP layer. Handlers are methods on it,
// so they reach the logger and store (and later: sessions...) without globals.
type Server struct {
	log        *slog.Logger
	store      Store
	devMode    bool
	uploadsDir string
	metrics    *metrics
	mailer     mail.Mailer
	publicURL  string
	google     googleOAuth
	limiter    *rateLimiter
}

// extraCollectors lets main contribute collectors that need dependencies the
// api layer doesn't own (e.g. the pgx pool stats collector).
func NewServer(log *slog.Logger, store Store, devMode bool, uploadsDir string,
	opts Options, extraCollectors ...prometheus.Collector) *Server {
	mailer := opts.Mailer
	if mailer == nil {
		mailer = &mail.LogSink{Log: log}
	}
	publicURL := strings.TrimRight(opts.PublicURL, "/")
	if publicURL == "" {
		publicURL = "http://localhost:5173"
	}
	return &Server{
		log:        log,
		store:      store,
		devMode:    devMode,
		uploadsDir: uploadsDir,
		metrics:    newMetrics(extraCollectors...),
		mailer:     mailer,
		publicURL:  publicURL,
		google: googleOAuth{
			clientID:     opts.GoogleClientID,
			clientSecret: opts.GoogleClientSecret,
			publicURL:    publicURL,
			authURL:      opts.GoogleAuthURL,
			tokenURL:     opts.GoogleTokenURL,
			userinfoURL:  opts.GoogleUserinfoURL,
		},
		limiter: newRateLimiter(),
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
		// ...and the market. After withLocale, because the currency
		// negotiation falls back to whatever the language negotiation
		// found — order matters here in a way it does not for the pair
		// above.
		r.Use(s.withCurrency)

		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)
		r.Post("/auth/logout", s.handleLogout)
		r.Get("/auth/me", s.handleMe)
		// E8: the reset flow (both anonymous by nature — the whole point is
		// the caller cannot sign in) and the Google flow.
		r.Post("/auth/forgot-password", s.handleForgotPassword)
		r.Post("/auth/reset-password", s.handleResetPassword)
		r.Get("/auth/oauth/google", s.handleGoogleStart)
		r.Get("/auth/oauth/google/callback", s.handleGoogleCallback)
		// E9: the newsletter — all anonymous by nature (the footer form and
		// two emailed-link pages). Confirm/unsubscribe are POSTs on purpose:
		// mail scanners prefetch GET links, and a mutating GET would let a
		// robot complete the double opt-in (see newsletter.go).
		r.Post("/newsletter/subscribe", s.handleNewsletterSubscribe)
		r.Post("/newsletter/confirm", s.handleNewsletterConfirm)
		r.Post("/newsletter/unsubscribe", s.handleNewsletterUnsubscribe)

		r.Get("/categories", s.handleListCategories)
		r.Get("/catalog/facets", s.handleCatalogFacets)
		r.Get("/products", s.handleListProducts)
		r.Get("/products/{slug}", s.handleGetProduct)
		r.Get("/products/{slug}/related", s.handleRelatedProducts)
		// Published reviews only — the handler PINS the status rather than
		// reading it from the query string, or `?status=pending` would
		// publish everything the moderator has not looked at yet.
		r.Get("/products/{slug}/reviews", s.handleListReviews)

		// Logged-in customers: cart and checkout.
		r.Group(func(r chi.Router) {
			r.Use(s.requireUser)
			r.Get("/cart", s.handleGetCart)
			r.Put("/cart/items", s.handleSetCartItem)
			r.Delete("/cart/items/{variantID}", s.handleDeleteCartItem)
			// E7: the promo box and the one calculator every money screen
			// reads. Preview is POST — a per-session computation, not an
			// addressable resource.
			r.Post("/cart/promo", s.handleApplyPromo)
			r.Delete("/cart/promo", s.handleRemovePromo)
			r.Post("/checkout/preview", s.handleCheckoutPreview)
			r.Post("/orders", s.handleCreateOrder)
			r.Get("/orders", s.handleListMyOrders)
			r.Get("/orders/{id}", s.handleGetOrder)
			// The saved address, for pre-filling the checkout form. Under
			// /account rather than /addresses because E8's account page is
			// its natural home and the URL should not have to move.
			r.Get("/account/address", s.handleGetDefaultAddress)
			// E8: the wishlist (set-semantics like the cart) and the
			// account page's address book.
			r.Get("/wishlist", s.handleListWishlist)
			r.Put("/wishlist/{productID}", s.handleAddWishlistItem)
			r.Delete("/wishlist/{productID}", s.handleRemoveWishlistItem)
			r.Post("/wishlist/save-for-later", s.handleSaveForLater)
			r.Get("/account/addresses", s.handleListAddresses)
			r.Post("/account/addresses", s.handleCreateAddress)
			r.Put("/account/addresses/{id}", s.handleUpdateAddress)
			r.Delete("/account/addresses/{id}", s.handleDeleteAddress)
			// Writing a review needs a session; the store additionally
			// requires a DELIVERED order containing the product.
			r.Post("/products/{slug}/reviews", s.handleCreateReview)
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

			// E3: the editorial half of a product page.
			r.Put("/products/{id}/images", s.handleSaveProductImages)
			r.Delete("/products/{id}/images/{imageID}", s.handleDeleteProductImage)
			r.Put("/products/{id}/editorial", s.handleSaveProductEditorial)
			r.Put("/products/{id}/related", s.handleSaveProductRelated)

			// E4: the moderation queue.
			r.Get("/reviews", s.handleAdminListReviews)
			r.Patch("/reviews/{id}", s.handleModerateReview)
		})
	})

	if s.devMode {
		// Deliberately slow endpoint for observing graceful shutdown
		// and client cancellation. Never registered in prod.
		r.Get("/debug/slow", s.handleSlow)
	}

	return r
}
