package api_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
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
}

func newFakeStore() *fakeStore {
	return &fakeStore{sessions: make(map[string]domain.User)}
}

// --- CategoryStore ---

func (f *fakeStore) ListCategories(_ context.Context) ([]domain.Category, error) {
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

func (f *fakeStore) ListProducts(_ context.Context, _ domain.ProductFilter) ([]domain.Product, int, error) {
	return f.products, len(f.products), nil
}

func (f *fakeStore) GetProductBySlug(_ context.Context, slug string) (domain.Product, error) {
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

func (f *fakeStore) GetCart(_ context.Context, _ int64) ([]domain.CartItem, error) {
	return nil, nil
}
func (f *fakeStore) SetCartItem(_ context.Context, _, _ int64, _ int) error   { return nil }
func (f *fakeStore) DeleteCartItem(_ context.Context, _, _ int64) error       { return nil }
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
	return api.NewServer(logger, fake, false).Routes()
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
