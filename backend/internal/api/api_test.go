package api_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

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

	// E5. lastCurrency is the currency middleware'"'"'s answer as the store saw
	// it — the only way a handler test can prove the ?currency / cookie /
	// Accept-Language chain reached the query rather than being dropped.
	lastCurrency domain.Currency
	cart         []domain.CartItem

	// E6 checkout.
	lastCheckout domain.CheckoutInput
	orders       []domain.Order
	// F2: who the status mailer finds when it looks up an order's customer.
	usersByID      map[int64]domain.User
	shippingRates  map[domain.Currency]domain.ShippingRate
	defaultAddress *domain.AddressEntry

	// A2 reorder: what the fake hands back, and which user/order the
	// handler asked for — the merge logic itself is the store suite's job.
	reorderResult domain.ReorderResult
	reorderUser   int64
	reorderOrder  int64

	// A5 settings: recording fields for the handler contracts.
	profileName        string
	profilePhone       string
	newHash            string
	keptToken          string
	notifyOrderUpdates *bool
	newsletterStatus   string
	unsubscribedEmail  string

	// E7 promotions. promos is the fake's promo_codes table (keyed by
	// NORMALIZED code); cartPromo the applied one; priorOrders the hive-club
	// fact; orderErr lets a test drive handleCreateOrder's promo_invalid
	// mapping without a database producing the condition.
	promos      map[string]domain.Promo
	cartPromo   *domain.Promo
	priorOrders int
	upsell      *domain.Upsell
	orderErr    error
	// F2: the admin CRUD's table — a slice, because the admin list is
	// ordered where the shopper's `promos` lookup map is not.
	adminPromos []domain.Promo

	// E8 accounts. Mostly recording fields, same philosophy as the E3
	// editorial fakes: handler tests prove requests become the right calls;
	// the store's own behaviour has the Docker-backed suite.
	wishlist        map[int64]bool // product id → hearted
	wishlistItems   []domain.WishlistItem
	savedForLater   int64
	resetUserID     int64
	resetToken      string
	consumedToken   string
	newPasswordHash string
	consumeErr      error
	addresses       []domain.AddressEntry
	deletedAddress  int64
	oauthUser       domain.User
	oauthProvider   string
	oauthSubject    string
	oauthEmail      string
	userByEmail     *domain.User

	// E9 newsletter.
	newsletterEmail        string
	newsletterToken        string
	newsletterLive         bool
	newsletterConfirmed    string
	newsletterUnsubscribed string
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		sessions:  make(map[string]domain.User),
		wishlist:  make(map[int64]bool),
		usersByID: make(map[int64]domain.User),
	}
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

// --- F2 category admin (decision #95): behaving fakes on the slice the
// create fake already fills, with the same rules the real store enforces
// through SQL — slug uniqueness, RESTRICT-style delete, positional reorder.

func (f *fakeStore) AdminCategories(_ context.Context) ([]domain.Category, error) {
	return f.categories, nil
}

func (f *fakeStore) UpdateCategory(_ context.Context, c *domain.Category) error {
	for _, existing := range f.categories {
		if existing.ID != c.ID && existing.Slug == c.Slug {
			return domain.ErrSlugTaken
		}
	}
	for i := range f.categories {
		if f.categories[i].ID == c.ID {
			c.CreatedAt = f.categories[i].CreatedAt
			f.categories[i] = *c
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) DeleteCategory(_ context.Context, id int64) error {
	for _, p := range f.products {
		if p.CategoryID == id {
			return domain.ErrCategoryInUse
		}
	}
	for i := range f.categories {
		if f.categories[i].ID == id {
			f.categories = append(f.categories[:i], f.categories[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) ReorderCategories(_ context.Context, ids []int64) error {
	for pos, id := range ids {
		found := false
		for i := range f.categories {
			if f.categories[i].ID == id {
				f.categories[i].SortOrder = (pos + 1) * 10
				found = true
			}
		}
		if !found {
			return domain.ErrNotFound
		}
	}
	return nil
}

// --- ProductStore ---

func (f *fakeStore) ListProducts(_ context.Context, filter domain.ProductFilter) ([]domain.Product, int, error) {
	f.lastLocale = filter.EffectiveLocale()
	f.lastFilter = filter
	return f.products, len(f.products), nil
}

func (f *fakeStore) ListProductSlugs(_ context.Context) ([]string, error) {
	slugs := make([]string, 0, len(f.products))
	for _, p := range f.products {
		slugs = append(slugs, p.Slug)
	}
	return slugs, nil
}

func (f *fakeStore) CatalogFacets(_ context.Context, filter domain.ProductFilter) (domain.CatalogFacets, error) {
	f.lastLocale = filter.EffectiveLocale()
	f.lastFilter = filter
	return f.facets, nil
}

func (f *fakeStore) GetProductBySlug(_ context.Context, slug string, view domain.View) (domain.Product, error) {
	f.lastLocale = view.EffectiveLocale()
	f.lastCurrency = view.EffectiveCurrency()
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

func (f *fakeStore) ListRelated(_ context.Context, _ string, view domain.View) ([]domain.Product, error) {
	f.lastLocale = view.EffectiveLocale()
	f.lastCurrency = view.EffectiveCurrency()
	return f.related, nil
}

func (f *fakeStore) ListCuratedRelated(_ context.Context, _ string, view domain.View) ([]domain.Product, error) {
	f.lastLocale = view.EffectiveLocale()
	f.lastCurrency = view.EffectiveCurrency()
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

func (f *fakeStore) UpdateVariant(_ context.Context, variantID int64, prices domain.Money, stockQty int) error {
	for i := range f.products {
		for j := range f.products[i].Variants {
			if f.products[i].Variants[j].ID == variantID {
				f.products[i].Variants[j].Prices = prices
				f.products[i].Variants[j].PriceMinor = prices[domain.DefaultCurrency]
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

func (f *fakeStore) GetUserByEmail(_ context.Context, email string) (domain.User, error) {
	// E8: settable, because the reset and remember-me tests need a user the
	// handler can FIND — the zero fake keeps its "nobody exists" behaviour.
	if f.userByEmail != nil && f.userByEmail.Email == email {
		return *f.userByEmail, nil
	}
	return domain.User{}, domain.ErrNotFound
}

// F2: the status mailer's read. A map, because the mail tests need users
// with DIFFERENT notify toggles side by side; empty map = nobody exists.
func (f *fakeStore) GetUserByID(_ context.Context, userID int64) (domain.User, error) {
	if u, ok := f.usersByID[userID]; ok {
		return u, nil
	}
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

func (f *fakeStore) GetCart(_ context.Context, _ int64, view domain.View) ([]domain.CartItem, error) {
	f.lastCurrency = view.EffectiveCurrency()
	return f.cart, nil
}
func (f *fakeStore) SetCartItem(_ context.Context, _, _ int64, _ int) error { return nil }
func (f *fakeStore) DeleteCartItem(_ context.Context, _, _ int64) error     { return nil }
func (f *fakeStore) CreateOrder(_ context.Context, _ int64, view domain.View, in domain.CheckoutInput) (domain.Order, error) {
	f.lastCurrency = view.EffectiveCurrency()
	f.lastCheckout = in
	if f.orderErr != nil {
		return domain.Order{}, f.orderErr
	}
	if len(f.cart) == 0 {
		return domain.Order{}, domain.ErrEmptyCart
	}
	return domain.Order{
		ID: 1, Status: domain.OrderPending,
		Currency: view.EffectiveCurrency(), Locale: view.EffectiveLocale(),
		ShipTo: &in.Address, PaymentMethod: in.PaymentMethod,
		PaymentStatus: domain.PaymentUnpaid,
	}, nil
}

// --- PromoStore (E7) ---

// F2 promo CRUD: a behaving fake, like its order siblings — the duplicate
// rule runs on the domain's own normalization, so handler tests can drive
// the 409 without duplicating store logic.

func (f *fakeStore) ListPromos(_ context.Context) ([]domain.Promo, error) {
	return f.adminPromos, nil
}

func (f *fakeStore) CreatePromo(_ context.Context, in domain.PromoInput) (domain.Promo, error) {
	code := domain.NormalizePromoCode(in.Code)
	for _, p := range f.adminPromos {
		if p.Code == code {
			return domain.Promo{}, domain.ErrPromoCodeTaken
		}
	}
	promo := promoFromInput(int64(len(f.adminPromos)+1), code, in)
	f.adminPromos = append(f.adminPromos, promo)
	return promo, nil
}

func (f *fakeStore) UpdatePromo(_ context.Context, id int64, in domain.PromoInput) (domain.Promo, error) {
	code := domain.NormalizePromoCode(in.Code)
	for _, p := range f.adminPromos {
		if p.ID != id && p.Code == code {
			return domain.Promo{}, domain.ErrPromoCodeTaken
		}
	}
	for i, p := range f.adminPromos {
		if p.ID == id {
			f.adminPromos[i] = promoFromInput(id, code, in)
			return f.adminPromos[i], nil
		}
	}
	return domain.Promo{}, domain.ErrNotFound
}

func promoFromInput(id int64, code string, in domain.PromoInput) domain.Promo {
	promo := domain.Promo{
		ID: id, Code: code, Kind: in.Kind,
		StartsAt: in.StartsAt, EndsAt: in.EndsAt,
		MaxRedemptions: in.MaxRedemptions, Active: in.Active,
		Values: in.Values,
	}
	if in.Percent != nil {
		promo.Percent = *in.Percent
	}
	return promo
}

func (f *fakeStore) PromoForUser(_ context.Context, code string, _ int64) (domain.Promo, error) {
	if p, ok := f.promos[domain.NormalizePromoCode(code)]; ok {
		return p, nil
	}
	return domain.Promo{}, domain.ErrNotFound
}

func (f *fakeStore) CartPromoForUser(_ context.Context, _ int64) (*domain.Promo, error) {
	return f.cartPromo, nil
}

func (f *fakeStore) SetCartPromo(_ context.Context, _ int64, codeID int64) error {
	for _, p := range f.promos {
		if p.ID == codeID {
			promo := p
			f.cartPromo = &promo
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) ClearCartPromo(_ context.Context, _ int64) error {
	f.cartPromo = nil
	return nil
}

func (f *fakeStore) PriorOrders(_ context.Context, _ int64) (int, error) {
	return f.priorOrders, nil
}

func (f *fakeStore) UpsellForGap(_ context.Context, _ domain.View, _ int64) (*domain.Upsell, error) {
	return f.upsell, nil
}

func (f *fakeStore) GetOrder(_ context.Context, orderID int64) (domain.Order, error) {
	for _, o := range f.orders {
		if o.ID == orderID {
			return o, nil
		}
	}
	return domain.Order{}, domain.ErrNotFound
}

func (f *fakeStore) ShippingRates(_ context.Context) (map[domain.Currency]domain.ShippingRate, error) {
	if f.shippingRates != nil {
		return f.shippingRates, nil
	}
	// The migration's bootstrap rows, so cart tests quote realistic money
	// without every test constructing a rate table.
	free := func(n int64) *int64 { return &n }
	return map[domain.Currency]domain.ShippingRate{
		domain.CurrencyUSD: {BaseMinor: 400, ColdChainSurchargeMinor: 600, FreeOverMinor: free(7000)},
		domain.CurrencyAMD: {BaseMinor: 1900, ColdChainSurchargeMinor: 2900, FreeOverMinor: free(33500)},
	}, nil
}

func (f *fakeStore) DefaultAddress(_ context.Context, _ int64) (domain.AddressEntry, error) {
	if f.defaultAddress == nil {
		return domain.AddressEntry{}, domain.ErrNotFound
	}
	return *f.defaultAddress, nil
}
func (f *fakeStore) ListOrdersByUser(_ context.Context, _ int64) ([]domain.Order, error) {
	return nil, nil
}

// Reorder mirrors the real store's contract: ownership answered here, a
// stranger's (or missing) order id → ErrNotFound.
func (f *fakeStore) Reorder(_ context.Context, userID, orderID int64) (domain.ReorderResult, error) {
	f.reorderUser, f.reorderOrder = userID, orderID
	for _, o := range f.orders {
		if o.ID == orderID {
			if o.UserID != userID {
				return domain.ReorderResult{}, domain.ErrNotFound
			}
			return f.reorderResult, nil
		}
	}
	return domain.ReorderResult{}, domain.ErrNotFound
}
func (f *fakeStore) ListAllOrders(_ context.Context) ([]domain.Order, error) { return nil, nil }

// CancelOrderByCustomer mirrors the real gates with the domain's own
// functions: ownership first (ErrNotFound hides existence), then the
// pending-only window.
func (f *fakeStore) CancelOrderByCustomer(_ context.Context, userID, orderID int64) (domain.Order, error) {
	for i, o := range f.orders {
		if o.ID == orderID {
			if o.UserID != userID {
				return domain.Order{}, domain.ErrNotFound
			}
			if !domain.CustomerMayCancelOrder(o.Status) {
				return domain.Order{}, domain.ErrTooLateToCancel
			}
			f.orders[i].Status = domain.OrderCancelled
			return f.orders[i], nil
		}
	}
	return domain.Order{}, domain.ErrNotFound
}

// UpdateOrderStatus graduated from stub to behaving fake when F2's status
// mailer gave handler tests something to observe after a 200 — same shape
// as its payment sibling below: the transition brain is the domain's, so
// the fake borrows it rather than duplicating it.
func (f *fakeStore) UpdateOrderStatus(_ context.Context, orderID int64, to string) (domain.Order, error) {
	for i, o := range f.orders {
		if o.ID == orderID {
			if !domain.ValidOrderTransition(o.Status, to) {
				return domain.Order{}, fmt.Errorf("%w: %s → %s", domain.ErrInvalidTransition, o.Status, to)
			}
			f.orders[i].Status = to
			return f.orders[i], nil
		}
	}
	return domain.Order{}, domain.ErrNotFound
}

// UpdateOrderPaymentStatus behaves rather than stubs: it asks the SAME
// domain function the real store asks, so handler tests can walk the whole
// 200/409 surface — without duplicating any SQL-shaped logic, because the
// machine lives in domain, not in the store.
func (f *fakeStore) UpdateOrderPaymentStatus(_ context.Context, orderID int64, to string) (domain.Order, error) {
	for i, o := range f.orders {
		if o.ID == orderID {
			if !domain.ValidPaymentTransition(o.PaymentStatus, to) {
				return domain.Order{}, fmt.Errorf("%w: payment %s → %s", domain.ErrInvalidTransition, o.PaymentStatus, to)
			}
			f.orders[i].PaymentStatus = to
			return f.orders[i], nil
		}
	}
	return domain.Order{}, domain.ErrNotFound
}

// --- AccountStore (E8) ---

func (f *fakeStore) ListWishlist(_ context.Context, _ int64, view domain.View) ([]domain.WishlistItem, error) {
	f.lastCurrency = view.EffectiveCurrency()
	return f.wishlistItems, nil
}

// AddWishlistToCart hands back a canned report, like Reorder — the merge
// logic is the store suite's job.
func (f *fakeStore) AddWishlistToCart(_ context.Context, _ int64) (domain.ReorderResult, error) {
	return f.reorderResult, nil
}

// --- A5 settings — recording fakes: handler tests prove requests become
// the right calls; the store's behaviour has the Docker-backed suite.

func (f *fakeStore) UpdateProfile(_ context.Context, _ int64, fullName, phone string) error {
	f.profileName, f.profilePhone = fullName, phone
	return nil
}

func (f *fakeStore) ChangePassword(_ context.Context, _ int64, newHash, keepToken string) error {
	f.newHash, f.keptToken = newHash, keepToken
	return nil
}

func (f *fakeStore) SetNotifyOrderUpdates(_ context.Context, _ int64, on bool) error {
	f.notifyOrderUpdates = &on
	return nil
}

func (f *fakeStore) NewsletterStatusByEmail(_ context.Context, _ string) (string, error) {
	if f.newsletterStatus == "" {
		return domain.NewsletterNone, nil
	}
	return f.newsletterStatus, nil
}

func (f *fakeStore) UnsubscribeNewsletterByEmail(_ context.Context, email string) error {
	f.unsubscribedEmail = email
	return nil
}

func (f *fakeStore) AddWishlistItem(_ context.Context, _ int64, productID int64) error {
	if !f.hasProduct(productID) {
		return domain.ErrNotFound
	}
	f.wishlist[productID] = true
	return nil
}

func (f *fakeStore) RemoveWishlistItem(_ context.Context, _ int64, productID int64) error {
	delete(f.wishlist, productID)
	return nil
}

func (f *fakeStore) SaveForLater(_ context.Context, _ int64, variantID int64) error {
	for _, it := range f.cart {
		if it.VariantID == variantID {
			f.savedForLater = variantID
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) CreatePasswordReset(_ context.Context, userID int64, token string, _ time.Time) error {
	f.resetUserID, f.resetToken = userID, token
	return nil
}

func (f *fakeStore) ConsumePasswordReset(_ context.Context, token, newHash string) error {
	if f.consumeErr != nil {
		return f.consumeErr
	}
	f.consumedToken, f.newPasswordHash = token, newHash
	return nil
}

func (f *fakeStore) ListAddresses(_ context.Context, _ int64) ([]domain.AddressEntry, error) {
	return f.addresses, nil
}

func (f *fakeStore) CreateAddress(_ context.Context, _ int64, e domain.AddressEntry) (domain.AddressEntry, error) {
	e.ID = int64(len(f.addresses) + 1)
	e.IsDefault = e.IsDefault || len(f.addresses) == 0
	f.addresses = append(f.addresses, e)
	return e, nil
}

func (f *fakeStore) UpdateAddress(_ context.Context, _ int64, e domain.AddressEntry) error {
	for i := range f.addresses {
		if f.addresses[i].ID == e.ID {
			f.addresses[i] = e
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) DeleteAddress(_ context.Context, _ int64, addressID int64) error {
	for i := range f.addresses {
		if f.addresses[i].ID == addressID {
			f.addresses = append(f.addresses[:i], f.addresses[i+1:]...)
			f.deletedAddress = addressID
			return nil
		}
	}
	return domain.ErrNotFound
}

func (f *fakeStore) FindOrCreateOAuthUser(_ context.Context, provider, subject, email string) (domain.User, error) {
	f.oauthProvider, f.oauthSubject, f.oauthEmail = provider, subject, email
	if f.oauthUser.ID != 0 {
		return f.oauthUser, nil
	}
	return domain.User{ID: 42, Email: email, Role: domain.RoleCustomer}, nil
}

// --- NewsletterStore (E9) ---

func (f *fakeStore) SubscribeNewsletter(_ context.Context, email, token string) (bool, error) {
	f.newsletterEmail, f.newsletterToken = email, token
	// alreadyLive simulates a confirmed subscriber: no confirmation needed.
	return !f.newsletterLive, nil
}

func (f *fakeStore) ConfirmNewsletter(_ context.Context, token string) error {
	if f.newsletterToken != "" && token != f.newsletterToken {
		return domain.ErrNotFound
	}
	f.newsletterConfirmed = token
	return nil
}

func (f *fakeStore) UnsubscribeNewsletter(_ context.Context, token string) error {
	if f.newsletterToken != "" && token != f.newsletterToken {
		return domain.ErrNotFound
	}
	f.newsletterUnsubscribed = token
	return nil
}

// --- test helpers ---

// newTestServer wires a real router (all real middleware!) around the fake.
func newTestServer(fake *fakeStore) http.Handler {
	return newTestServerOpts(fake, api.Options{})
}

// newTestServerOpts is newTestServer with E8's knobs — the mailer fake and
// the endpoints of a pretend Google.
func newTestServerOpts(fake *fakeStore, opts api.Options) http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil)) // silent in tests
	return api.NewServer(logger, fake, false, os.TempDir(), opts).Routes()
}

// testPasswordHash bcrypts at MinCost — these tests measure handlers, not
// key-stretching.
func testPasswordHash(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
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
