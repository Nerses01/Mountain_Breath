package api

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

type variantResponse struct {
	ID    int64  `json:"id"`
	SKU   string `json:"sku"`
	Label string `json:"label"`

	// PriceMinor is the price in the currency this request resolved to, in
	// THAT currency's minor units — so 1400 is $14.00 and 5460 is 5,460 ֏.
	// A client that divides by 100 is now wrong half the time; see
	// frontend/src/lib/currencies.ts.
	//
	// Kept alongside Prices rather than replaced by it because it names the
	// one number the rest of the page computes with (line totals, the
	// "from" price, the sort the server already applied). Prices is for
	// DISPLAY — the design's muted second line.
	PriceMinor int64 `json:"price_minor"`
	// Prices is the same variant in every market the shop serves, keyed by
	// ISO code. Not derived from PriceMinor by a rate: per-market prices are
	// set independently, which is the whole point of variant_prices.
	Prices map[domain.Currency]int64 `json:"prices"`

	StockQty int `json:"stock_qty"`
}

// benefitResponse is one "Good for" chip. Slug travels alongside name
// because the client puts the slug in the query string when the chip is
// clicked — the name is for reading, the slug is for linking.
type benefitResponse struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// cardImageResponse is a photo as a CARD sees it: url and alt, nothing else.
// Hero first by the store's ordering, so [0] is what a static card renders
// and the rest exist for the hover slideshow. The video never rides here —
// a grid hover must not start a multi-megabyte download.
type cardImageResponse struct {
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type productResponse struct {
	ID          int64               `json:"id"`
	CategoryID  int64               `json:"category_id"`
	Slug        string              `json:"slug"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Images      []cardImageResponse `json:"images"`
	CreatedAt   time.Time           `json:"created_at"`
	Variants    []variantResponse   `json:"variants"`

	// The denormalized review aggregate (migration 000015). On the CARD as
	// well as the detail, because the design puts ★★★★★ on both — which is
	// exactly why it is a stored column rather than an aggregate query.
	RatingAvg   float64 `json:"rating_avg"`
	RatingCount int     `json:"rating_count"`

	// The category, resolved into the requested language. category_id alone
	// would force every client to fetch /categories and join by hand — and
	// to get the fallback chain right a second time.
	CategorySlug string `json:"category_slug"`
	CategoryName string `json:"category_name"`

	// Badge is a KEY the client looks up in its message catalogue
	// ("best_seller"), empty when the product has no badge; BadgeTone is how
	// to paint it. Same codes-not-prose contract as validation errors.
	Badge     string `json:"badge"`
	BadgeTone string `json:"badge_tone"`

	Benefits []benefitResponse `json:"benefits"`

	// Currency names what price_minor is denominated in — on the PRODUCT and
	// not only on the envelope, so a card that has been passed around a
	// component tree still knows how to render itself. Cheap: one three-byte
	// string per product.
	Currency domain.Currency `json:"currency"`

	// products.sales_count is deliberately NOT here. It orders the list
	// server-side, and that is all a shopper needs from it — publishing it
	// would tell every visitor (and every competitor) exactly how many jars
	// the family sells. The sort works fine without the number being
	// readable, which is the test for whether a field belongs in a response.
}

// --- Detail-only shapes (E3) ------------------------------------------
//
// A separate struct EMBEDDING the list shape rather than more optional
// fields on it. The two endpoints answer different questions, and a single
// struct would have made every card in the grid carry six nulls and every
// TypeScript consumer guess which fields are populated when.

type imageResponse struct {
	ID        int64  `json:"id"`
	URL       string `json:"url"`
	Alt       string `json:"alt"`
	IsPrimary bool   `json:"is_primary"`
}

type highlightResponse struct {
	Text string `json:"text"`
}

type usageCardResponse struct {
	Kicker string `json:"kicker"`
	Title  string `json:"title"`
	Body   string `json:"body"`
}

// videoResponse is the single video slot. No is_primary — the hero is
// always a photo (a CHECK in 000026 enforces it) — and Alt is the clip's
// accessible label, same translations table as a photo's alt.
type videoResponse struct {
	ID  int64  `json:"id"`
	URL string `json:"url"`
	Alt string `json:"alt"`
}

type productDetailResponse struct {
	productResponse

	Images []imageResponse `json:"images"`
	// A pointer so an absent video serializes as JSON null rather than a
	// zero-valued object the client would have to probe field by field —
	// one kind of absence, same rule as ship_to on an order.
	Video      *videoResponse      `json:"video"`
	Highlights []highlightResponse `json:"highlights"`
	UsageCards []usageCardResponse `json:"usage_cards"`

	Disclaimer   string `json:"disclaimer"`
	StorageNote  string `json:"storage_note"`
	HarvestNote  string `json:"harvest_note"`
	ShippingNote string `json:"shipping_note"`
	LabBatch     string `json:"lab_batch"`
	IsColdChain  bool   `json:"is_cold_chain"`

	// Answers "may the current viewer leave a review?", so the UI does not
	// reimplement the verified-purchase rule and guess. Always false for an
	// anonymous visitor. It is a HINT for rendering — the write path checks
	// the rule again, because anyone can POST.
	CanReview bool `json:"can_review"`
}

func toProductDetailResponse(p domain.Product, currency domain.Currency) productDetailResponse {
	images := make([]imageResponse, 0, len(p.Images))
	for _, img := range p.Images {
		images = append(images, imageResponse{
			ID: img.ID, URL: img.URL, Alt: img.Alt, IsPrimary: img.IsPrimary,
		})
	}
	highlights := make([]highlightResponse, 0, len(p.Highlights))
	for _, h := range p.Highlights {
		highlights = append(highlights, highlightResponse{Text: h.Text})
	}
	cards := make([]usageCardResponse, 0, len(p.UsageCards))
	for _, c := range p.UsageCards {
		cards = append(cards, usageCardResponse{Kicker: c.Kicker, Title: c.Title, Body: c.Body})
	}
	var video *videoResponse
	if p.Video != nil {
		video = &videoResponse{ID: p.Video.ID, URL: p.Video.URL, Alt: p.Video.Alt}
	}

	return productDetailResponse{
		productResponse: toProductResponse(p, currency),
		Images:          images,
		Video:           video,
		Highlights:      highlights,
		UsageCards:      cards,
		Disclaimer:      p.Disclaimer,
		StorageNote:     p.StorageNote,
		HarvestNote:     p.HarvestNote,
		ShippingNote:    p.ShippingNote,
		LabBatch:        p.LabBatch,
		IsColdChain:     p.IsColdChain,
		CanReview:       p.CanReview,
	}
}

// paginated is a generic envelope for list endpoints — the same shape for
// products now, orders later.
type paginated[T any] struct {
	Items   []T `json:"items"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func toProductResponse(p domain.Product, currency domain.Currency) productResponse {
	variants := make([]variantResponse, 0, len(p.Variants))
	for _, v := range p.Variants {
		variants = append(variants, variantResponse{
			ID: v.ID, SKU: v.SKU, Label: v.Label,
			PriceMinor: v.PriceMinor, Prices: v.Prices, StockQty: v.StockQty,
		})
	}
	benefits := make([]benefitResponse, 0, len(p.Benefits))
	for _, b := range p.Benefits {
		benefits = append(benefits, benefitResponse{Slug: b.Slug, Name: b.Name})
	}
	cardImages := make([]cardImageResponse, 0, len(p.Images))
	for _, img := range p.Images {
		cardImages = append(cardImages, cardImageResponse{URL: img.URL, Alt: img.Alt})
	}
	return productResponse{
		ID: p.ID, CategoryID: p.CategoryID, Slug: p.Slug, Name: p.Name,
		Description: p.Description, Images: cardImages,
		CreatedAt: p.CreatedAt, Variants: variants,
		CategorySlug: p.CategorySlug, CategoryName: p.CategoryName,
		RatingAvg: p.Rating.Average, RatingCount: p.Rating.Count,
		Badge: p.Badge, BadgeTone: p.BadgeTone, Benefits: benefits,
		Currency: currency,
	}
}

// productFilterFromQuery reads the shared catalog filter out of the query
// string. Shared by the listing and the facets endpoint on purpose: two
// parsers would eventually disagree about what "?benefit=energy&max_price=2000"
// means, and the sidebar counts would stop matching the grid.
//
// Everything here is defensive. A query string is the most public input the
// app has — anyone can type one — so every field either parses or falls back
// to a sane default. Nothing rejects the request: a shopper who edits the
// URL badly should see the shop, not a 400.
func productFilterFromQuery(r *http.Request) domain.ProductFilter {
	q := r.URL.Query()

	f := domain.ProductFilter{
		CategorySlug: q.Get("category"),
		Search:       q.Get("q"),
		Locale:       localeFromContext(r.Context()),
		// The price bounds below are in THIS currency's minor units, which
		// is why it has to be parsed before them: ?max_price=2000 means $20
		// in one market and 2,000 ֏ in another.
		Currency: currencyFromContext(r.Context()),
		// Repeated params, not a comma-separated list: ?benefit=energy&
		// benefit=skin. url.Values is already a map to a SLICE, so this is
		// what the standard library hands us for free, and it needs no
		// escaping rule for a benefit slug that contains a comma.
		BenefitSlugs:  q["benefit"],
		PriceMinMinor: int64QueryParam(q.Get("min_price")),
		PriceMaxMinor: int64QueryParam(q.Get("max_price")),
	}
	// An unknown sort falls back to the default rather than erroring — the
	// second return value of ParseProductSort is ignored deliberately here,
	// and the whitelist is what protects the ORDER BY.
	f.Sort, _ = domain.ParseProductSort(q.Get("sort"))

	// A reversed range ($30–$10) is a slider bug or a hand-edited URL, and
	// it would return nothing at all. Swapping is friendlier than an empty
	// grid and cannot surprise anyone.
	if f.PriceMinMinor != nil && f.PriceMaxMinor != nil && *f.PriceMinMinor > *f.PriceMaxMinor {
		f.PriceMinMinor, f.PriceMaxMinor = f.PriceMaxMinor, f.PriceMinMinor
	}
	return f
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := productFilterFromQuery(r)
	filter.Page = intQueryParam(q.Get("page"), 1)
	filter.PerPage = intQueryParam(q.Get("per_page"), 20)
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 20
	}

	products, total, err := s.store.ListProducts(r.Context(), filter)
	if err != nil {
		s.log.Error("listing products", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	items := make([]productResponse, 0, len(products))
	for _, p := range products {
		items = append(items, toProductResponse(p, filter.EffectiveCurrency()))
	}
	s.respondJSON(w, http.StatusOK, paginated[productResponse]{
		Items: items, Page: filter.Page, PerPage: filter.PerPage, Total: total,
	})
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	view := viewFromContext(r.Context())
	p, err := s.store.GetProductBySlug(r.Context(), slug, view)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such product")
			return
		}
		s.log.Error("getting product", "slug", slug, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// Only asked for a signed-in viewer: an anonymous one cannot review
	// anything, so the query would be a round trip whose answer is already
	// known. A failure here is logged and swallowed rather than 500ing the
	// page — this decides whether a form is rendered, and a product page is
	// worth more than a review box.
	if user, ok := userFrom(r.Context()); ok {
		can, err := s.store.CanReview(r.Context(), user.ID, slug)
		if err != nil {
			s.log.Error("checking review eligibility", "slug", slug, "error", err)
		}
		p.CanReview = can
	}

	s.respondJSON(w, http.StatusOK, toProductDetailResponse(p, view.EffectiveCurrency()))
}

// GET /products/{slug}/related — "Often taken together".
//
// Its own endpoint rather than a field on the detail response: the panel sits
// below the fold, changes far less often than stock or price, and a client
// that only needs the buy box should not pay for four extra products. It also
// lets the frontend cache the two under different keys.
func (s *Server) handleRelatedProducts(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	view := viewFromContext(r.Context())

	// ?curated=true asks ONLY for what the admin curated, with no computed
	// fallback. The admin's picker needs that distinction and the storefront
	// does not: pre-filling from the normal read would present the COMPUTED
	// list as though it were curated, and saving it would silently freeze a
	// dynamic panel into a static one.
	var (
		products []domain.Product
		err      error
	)
	if r.URL.Query().Get("curated") == "true" {
		products, err = s.store.ListCuratedRelated(r.Context(), slug, view)
	} else {
		products, err = s.store.ListRelated(r.Context(), slug, view)
	}
	if err != nil {
		s.log.Error("listing related products", "slug", slug, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	// An unknown slug returns an empty list, not a 404: this is a panel on a
	// page, and the page itself already 404s if the product is missing.
	items := make([]productResponse, 0, len(products))
	for _, p := range products {
		items = append(items, toProductResponse(p, view.EffectiveCurrency()))
	}
	s.respondJSON(w, http.StatusOK, items)
}

// intQueryParam parses s, falling back on absence or garbage. Query params
// are user input — never trust, always default.
func intQueryParam(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

// int64QueryParam is intQueryParam for an OPTIONAL number: absent or
// unparseable both mean "no bound", which is nil rather than a magic value.
// Negative bounds are dropped too — money is never negative here, and a
// price floor of -5 is a client bug, not a filter.
func int64QueryParam(s string) *int64 {
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return nil
	}
	return &n
}
