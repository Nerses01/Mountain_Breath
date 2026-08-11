package api_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/Nerses01/Mountain_Breath/backend/internal/api"
	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// fakeStore satisfies api.Store entirely in memory — this is why the Store
// interfaces are defined at the consumer: handler tests need no database,
// no Docker, no network, and run in microseconds.
type fakeStore struct {
	categories []domain.Category
	products   []domain.Product
	// session token → user, simulating the sessions table
	sessions map[string]domain.User
	// lastLocale records what the handler passed down, so a test can prove
	// the negotiated language actually reaches the store rather than being
	// resolved and then dropped.
	lastLocale domain.Locale
	// lastFilter does the same job for the whole catalog filter: parsing a
	// query string into a ProductFilter is API-layer logic, and this is what
	// lets it be tested without a database behind it.
	lastFilter          domain.ProductFilter
	facets              domain.CatalogFacets
	related             []domain.Product
	curatedRelated      []domain.Product
	curatedRelatedAsked bool

	// What the E3 editorial handlers passed down. Recorded rather than
	// stored: these tests are about request bodies becoming the right calls;
	// the store's own behaviour is covered by the Docker-backed suite.
	savedImages       []domain.ProductImage
	lastImageAlts     map[domain.Locale]string
	lastImageAltsByID map[int64]map[domain.Locale]string
	deletedImageID    int64
	savedEditorial    map[domain.Locale]domain.ProductEditorial
	savedRelatedIDs   []int64

	// E4 reviews. reviewErr lets a test drive the handler's error mapping
	// (403 for a non-purchaser, 409 for a second review) without a database
	// to produce those conditions naturally.
	reviews          []domain.Review
	lastReviewFilter domain.ReviewFilter
	createdReview    domain.Review
	reviewErr        error
	moderatedID      int64
	moderatedStatus  string
	canReview        bool
}

func newFakeStore() *fakeStore {
	return &fakeStore{sessions: make(map[string]domain.User)}
}

// --- CategoryStore ---

func (f *fakeStore) ListCategories(_ context.Context, locale domain.Locale) ([]domain.Category, error) {
	f.lastLocale = locale
	return f.categories, nil
}

func (f *fakeStore) CreateCategory(_ context.Context, c *domain.Category) error {
	for _, existing := range f.categories {
		if existing.Slug == c.Slug {
			return domain.ErrSlugTaken
		}
	}
	c.ID = int64(len(f.categories) + 1)
	c.CreatedAt = time.Now()
	f.categories = append(f.categories, *c)
	return nil
}

// --- ProductStore ---

func (f *fakeStore) ListProducts(_ context.Context, filter domain.ProductFilter) ([]domain.Product, int, error) {
	f.lastLocale = filter.EffectiveLocale()
	f.lastFilter = filter
	return f.products, len(f.products), nil
}

func (f *fakeStore) CatalogFacets(_ context.Context, filter domain.ProductFilter) (domain.CatalogFacets, error) {
	f.lastLocale = filter.EffectiveLocale()
	f.lastFilter = filter
	return f.facets, nil
}

func (f *fakeStore) GetProductBySlug(_ context.Context, slug string, locale domain.Locale) (domain.Product, error) {
	f.lastLocale = locale
	for _, p := range f.products {
		if p.Slug == slug {
			return p, nil
		}
	}
	return domain.Product{}, domain.ErrNotFound
}

func (f *fakeStore) CreateProduct(_ context.Context, p *domain.Product) error {
	for _, existing := range f.products {
		if existing.Slug == p.Slug {
			return domain.ErrSlugTaken
		}
		for _, v := range existing.Variants {
			for _, nv := range p.Variants {
				if v.SKU == nv.SKU {
					return domain.ErrSKUTaken
				}
			}
		}
	}
	p.ID = int64(len(f.products) + 1)
	f.products = append(f.products, *p)
	return nil
}

func (f *fakeStore) UpdateProduct(_ context.Context, p *domain.Product) error {
	for i := range f.products {
		if f.products[i].ID == p.ID {
			f.products[i] = *p
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) UpdateProductImage(_ context.Context, productID int64, imageURL string) error {
	for i := range f.products {
		if f.products[i].ID == productID {
			f.products[i].ImageURL = imageURL
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) ListRelated(_ context.Context, _ string, locale domain.Locale) ([]domain.Product, error) {
	f.lastLocale = locale
	return f.related, nil
}

func (f *fakeStore) ListCuratedRelated(_ context.Context, _ string, locale domain.Locale) ([]domain.Product, error) {
	f.lastLocale = locale
	f.curatedRelatedAsked = true
	return f.curatedRelated, nil
}

// --- E3 editorial writes ---
//
// These record their arguments rather than simulating storage: what the
// handler tests actually check is that a request body becomes the right
// call, since the store's own behaviour has integration tests behind Docker.

func (f *fakeStore) AddProductImage(_ context.Context, productID int64, url string, alts map[domain.Locale]string) (domain.ProductImage, error) {
	// The real store gets this from a foreign-key violation; the fake has to
	// check by hand, or the upload handler's "unknown product ⇒ 404, and no
	// orphan file left on disk" path is never exercised.
	if !f.hasProduct(productID) {
		return domain.ProductImage{}, domain.ErrNotFound
	}
	img := domain.ProductImage{ID: int64(len(f.savedImages) + 1), URL: url, IsPrimary: len(f.savedImages) == 0}
	f.savedImages = append(f.savedImages, img)
	f.lastImageAlts = alts
	return img, nil
}

func (f *fakeStore) hasProduct(id int64) bool {
	for _, p := range f.products {
		if p.ID == id {
			return true
		}
	}
	return false
}

func (f *fakeStore) SaveProductImages(_ context.Context, _ int64, images []domain.ProductImage, alts map[int64]map[domain.Locale]string) error {
	f.savedImages = images
	f.lastImageAltsByID = alts
	return nil
}

func (f *fakeStore) DeleteProductImage(_ context.Context, _, imageID int64) error {
	f.deletedImageID = imageID
	return nil
}

func (f *fakeStore) SaveProductEditorial(_ context.Context, _ int64, byLocale map[domain.Locale]domain.ProductEditorial) error {
	f.savedEditorial = byLocale
	return nil
}

func (f *fakeStore) SaveProductRelated(_ context.Context, _ int64, relatedIDs []int64) error {
	f.savedRelatedIDs = relatedIDs
	return nil
}

func (f *fakeStore) UpdateVariant(_ context.Context, variantID, priceMinor int64, stockQty int) error {
	for i := range f.products {
		for j := range f.products[i].Variants {
			if f.products[i].Variants[j].ID == variantID {
				f.products[i].Variants[j].PriceMinor = priceMinor
				f.products[i].Variants[j].StockQty = stockQty
				return nil
			}
		}
	}
	return domain.ErrNotFound
}

// --- ReviewStore ---

func (f *fakeStore) ListReviews(_ context.Context, filter domain.ReviewFilter) ([]domain.Review, int, error) {
	f.lastReviewFilter = filter
	return f.reviews, len(f.reviews), nil
}

func (f *fakeStore) CreateReview(_ context.Context, r *domain.Review, _ string) error {
	if f.reviewErr != nil {
		return f.reviewErr
	}
	r.ID = int64(len(f.reviews) + 1)
	r.Status = domain.ReviewPending
	f.createdReview = *r
	return nil
}

func (f *fakeStore) UpdateReviewStatus(_ context.Context, reviewID int64, status string) (domain.Review, error) {
	if f.reviewErr != nil {
		return domain.Review{}, f.reviewErr
	}
	f.moderatedID, f.moderatedStatus = reviewID, status
	return domain.Review{ID: reviewID, Status: status, Rating: 5}, nil
}

func (f *fakeStore) CanReview(_ context.Context, _ int64, _ string) (bool, error) {
	return f.canReview, nil
}

// --- UserStore / SessionStore (only what these tests exercise) ---

func (f *fakeStore) CreateUser(_ context.Context, u *domain.User) error {
	u.ID = 1
	return nil
}

func (f *fakeStore) GetUserByEmail(_ context.Context, _ string) (domain.User, error) {
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeStore) CreateSession(_ context.Context, token string, userID int64, _ time.Time) error {
	f.sessions[token] = domain.User{ID: userID}
	return nil
}

func (f *fakeStore) GetUserBySession(_ context.Context, token string) (domain.User, error) {
	if u, ok := f.sessions[token]; ok {
		return u, nil
	}
	return domain.User{}, domain.ErrNotFound
}

func (f *fakeStore) DeleteSession(_ context.Context, token string) error {
	delete(f.sessions, token)
	return nil
}

// --- CartStore / OrderStore (unused by these tests) ---

func (f *fakeStore) GetCart(_ context.Context, _ int64, _ domain.Locale) ([]domain.CartItem, error) {
	return nil, nil
}
func (f *fakeStore) SetCartItem(_ context.Context, _, _ int64, _ int) error { return nil }
func (f *fakeStore) DeleteCartItem(_ context.Context, _, _ int64) error     { return nil }
func (f *fakeStore) CreateOrder(_ context.Context, _ int64) (domain.Order, error) {
	return domain.Order{}, domain.ErrEmptyCart
}
func (f *fakeStore) ListOrdersByUser(_ context.Context, _ int64) ([]domain.Order, error) {
	return nil, nil
}
func (f *fakeStore) ListAllOrders(_ context.Context) ([]domain.Order, error) { return nil, nil }
func (f *fakeStore) UpdateOrderStatus(_ context.Context, _ int64, _ string) (domain.Order, error) {
	return domain.Order{}, domain.ErrNotFound
}

// --- test helpers ---

// newTestServer wires a real router (all real middleware!) around the fake.
func newTestServer(fake *fakeStore) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // silent in tests
	return api.NewServer(logger, fake, false, os.TempDir()).Routes()
}

// loginAs plants a session directly in the fake and returns its cookie.
func loginAs(fake *fakeStore, user domain.User) *http.Cookie {
	token := "test-token-" + user.Role
	fake.sessions[token] = user
	return &http.Cookie{Name: "mb_session", Value: token}
}

func doRequest(h http.Handler, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}
