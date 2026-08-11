package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

type newVariantRequest struct {
	SKU   string `json:"sku"`
	Label string `json:"label"`
	// One price per market, keyed by ISO code: {"USD": 1400, "AMD": 6700}.
	// The base currency is required; anything else is optional and falls
	// back to a converted price. Replaced the scalar price_minor in E5 —
	// a BREAKING change to this endpoint, and deliberately so: silently
	// accepting the old field would have written a dollar figure into
	// whatever market happened to be default.
	Prices   map[domain.Currency]int64 `json:"prices"`
	StockQty int                       `json:"stock_qty"`
}

// productTextRequest is one language's copy of a product. Same shape for
// create and update, and the same shape the admin form posts per locale tab.
type productTextRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`

	// E3's per-language notes. Optional and additive, so every client and
	// Postman request written before E3 keeps working unchanged — the same
	// rule the `translations` key itself followed when E1.5 added it.
	Disclaimer   string `json:"disclaimer"`
	StorageNote  string `json:"storage_note"`
	HarvestNote  string `json:"harvest_note"`
	ShippingNote string `json:"shipping_note"`
}

type createProductRequest struct {
	CategoryID int64  `json:"category_id"`
	Slug       string `json:"slug"`
	// Name/Description are the English copy — required, because every other
	// language falls back to them.
	Name        string              `json:"name"`
	Description string              `json:"description"`
	ImageURL    string              `json:"image_url"`
	Variants    []newVariantRequest `json:"variants"`
	// Optional, non-default locales only: {"hy": {"name": "...", ...}}.
	// Additive, so existing clients are unaffected.
	Translations map[string]productTextRequest `json:"translations"`

	// English copy of E3's notes; the other languages ride in Translations.
	Disclaimer   string `json:"disclaimer"`
	StorageNote  string `json:"storage_note"`
	HarvestNote  string `json:"harvest_note"`
	ShippingNote string `json:"shipping_note"`
	// Locale-invariant metadata (migration 000013).
	LabBatch    string `json:"lab_batch"`
	IsColdChain bool   `json:"is_cold_chain"`
}

type updateProductRequest struct {
	CategoryID   int64                         `json:"category_id"`
	Name         string                        `json:"name"`
	Description  string                        `json:"description"`
	ImageURL     string                        `json:"image_url"`
	IsActive     bool                          `json:"is_active"`
	Translations map[string]productTextRequest `json:"translations"`

	Disclaimer   string `json:"disclaimer"`
	StorageNote  string `json:"storage_note"`
	HarvestNote  string `json:"harvest_note"`
	ShippingNote string `json:"shipping_note"`
	LabBatch     string `json:"lab_batch"`
	IsColdChain  bool   `json:"is_cold_chain"`
}

// toDomainTranslations converts the wire shape into the domain one.
func toDomainTranslations(in map[string]productTextRequest) map[domain.Locale]domain.ProductText {
	parsed := parseLocaleMap(in)
	if parsed == nil {
		return nil
	}
	out := make(map[domain.Locale]domain.ProductText, len(parsed))
	for locale, t := range parsed {
		out[locale] = domain.ProductText{
			Name: t.Name, Description: t.Description,
			Disclaimer: t.Disclaimer, StorageNote: t.StorageNote,
			HarvestNote: t.HarvestNote, ShippingNote: t.ShippingNote,
		}
	}
	return out
}

type updateVariantRequest struct {
	// The DESIRED STATE of this variant's prices, not a patch: a currency
	// left out is removed, which is how an admin puts a variant back on the
	// converted fallback. See store.UpdateVariant.
	Prices   map[domain.Currency]int64 `json:"prices"`
	StockQty int                       `json:"stock_qty"`
}

// adminProductResponse extends the public shape with admin-only fields.
type adminProductResponse struct {
	productResponse
	IsActive bool `json:"is_active"`
}

func toAdminProductResponse(p domain.Product, currency domain.Currency) adminProductResponse {
	return adminProductResponse{productResponse: toProductResponse(p, currency), IsActive: p.IsActive}
}

// GET /admin/products — like the public list, but includes inactive products.
func (s *Server) handleAdminListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := domain.ProductFilter{
		CategorySlug:    q.Get("category"),
		Search:          q.Get("q"),
		IncludeInactive: true,
		Page:            intQueryParam(q.Get("page"), 1),
		PerPage:         intQueryParam(q.Get("per_page"), 50),
		Locale:          localeFromContext(r.Context()),
		Currency:        currencyFromContext(r.Context()),
	}
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 50
	}

	products, total, err := s.store.ListProducts(r.Context(), filter)
	if err != nil {
		s.log.Error("admin listing products", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	items := make([]adminProductResponse, 0, len(products))
	for _, p := range products {
		items = append(items, toAdminProductResponse(p, filter.EffectiveCurrency()))
	}
	s.respondJSON(w, http.StatusOK, paginated[adminProductResponse]{
		Items: items, Page: filter.Page, PerPage: filter.PerPage, Total: total,
	})
}

// POST /admin/products — create a product with its variants.
func (s *Server) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req createProductRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	product := domain.Product{
		CategoryID:   req.CategoryID,
		Slug:         req.Slug,
		Name:         req.Name,
		Description:  req.Description,
		ImageURL:     req.ImageURL,
		IsActive:     true,
		Translations: toDomainTranslations(req.Translations),
		Disclaimer:   req.Disclaimer,
		StorageNote:  req.StorageNote,
		HarvestNote:  req.HarvestNote,
		ShippingNote: req.ShippingNote,
		LabBatch:     req.LabBatch,
		IsColdChain:  req.IsColdChain,
	}
	for _, v := range req.Variants {
		product.Variants = append(product.Variants, domain.ProductVariant{
			SKU: v.SKU, Label: v.Label, Prices: v.Prices, StockQty: v.StockQty,
		})
	}

	if fields := domain.ValidateProduct(product); len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	if err := s.store.CreateProduct(r.Context(), &product); err != nil {
		s.respondProductError(w, err)
		return
	}
	s.respondJSON(w, http.StatusCreated, toAdminProductResponse(product, currencyFromContext(r.Context())))
}

// PUT /admin/products/{id} — full update of mutable fields (slug immutable:
// it is a public URL).
func (s *Server) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "product id must be a number")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req updateProductRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	translations := toDomainTranslations(req.Translations)

	// These two were still literal "required" strings after the switch to
	// validation codes — the constant makes that impossible to miss again.
	fields := make(map[string]string)
	if req.Name == "" {
		fields["name"] = domain.ValidationRequired
	}
	if req.CategoryID <= 0 {
		fields["category_id"] = domain.ValidationRequired
	}
	for locale, text := range translations {
		if text.Name == "" {
			fields["translations."+string(locale)+".name"] = domain.ValidationRequired
		}
	}
	for k, v := range domain.ValidateTranslationLocales("translations", translations) {
		fields[k] = v
	}
	if len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	product := domain.Product{
		ID: id, CategoryID: req.CategoryID, Name: req.Name,
		Description: req.Description, ImageURL: req.ImageURL, IsActive: req.IsActive,
		Translations: translations,
		Disclaimer:   req.Disclaimer,
		StorageNote:  req.StorageNote,
		HarvestNote:  req.HarvestNote,
		ShippingNote: req.ShippingNote,
		LabBatch:     req.LabBatch,
		IsColdChain:  req.IsColdChain,
	}
	if err := s.store.UpdateProduct(r.Context(), &product); err != nil {
		s.respondProductError(w, err)
		return
	}
	s.respondJSON(w, http.StatusOK, toAdminProductResponse(product, currencyFromContext(r.Context())))
}

// PATCH /admin/variants/{id} — prices and stock, the fields that change in
// daily shop operation.
func (s *Server) handleUpdateVariant(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_id", "variant id must be a number")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req updateVariantRequest
	if err := dec.Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON: "+err.Error())
		return
	}

	// Same rules as ValidateProduct applies to a new variant, restated for
	// one variant: the base price must be there and positive, the others
	// must at least be currencies the shop knows.
	fields := make(map[string]string)
	if req.Prices[domain.BaseCurrency] <= 0 {
		fields["prices."+string(domain.BaseCurrency)] = domain.ValidationPositive
	}
	for c, minor := range req.Prices {
		if _, ok := domain.ParseCurrency(string(c)); !ok {
			fields["prices."+string(c)] = domain.ValidationUnknownCurrency
			continue
		}
		if minor <= 0 {
			fields["prices."+string(c)] = domain.ValidationPositive
		}
	}
	if req.StockQty < 0 {
		fields["stock_qty"] = domain.ValidationNotNegative
	}
	if len(fields) > 0 {
		s.respondValidationError(w, fields)
		return
	}

	if err := s.store.UpdateVariant(r.Context(), id, req.Prices, req.StockQty); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such variant")
			return
		}
		s.log.Error("updating variant", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// respondProductError maps the product-domain sentinels to HTTP once, for
// both create and update paths.
func (s *Server) respondProductError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrSlugTaken):
		s.respondError(w, http.StatusConflict, "slug_taken", "a product with this slug already exists")
	case errors.Is(err, domain.ErrSKUTaken):
		s.respondError(w, http.StatusConflict, "sku_taken", "a variant with this SKU already exists")
	case errors.Is(err, domain.ErrVariantLabelTaken):
		s.respondError(w, http.StatusConflict, "variant_label_taken", "two variants of one product cannot share a label")
	case errors.Is(err, domain.ErrCategoryNotFound):
		s.respondValidationError(w, map[string]string{"category_id": "no such category"})
	case errors.Is(err, domain.ErrNotFound):
		s.respondError(w, http.StatusNotFound, "not_found", "no such product")
	default:
		s.log.Error("product write", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
