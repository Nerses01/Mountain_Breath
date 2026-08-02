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
	ID         int64  `json:"id"`
	SKU        string `json:"sku"`
	Label      string `json:"label"`
	PriceMinor int64  `json:"price_minor"`
	StockQty   int    `json:"stock_qty"`
}

type productResponse struct {
	ID          int64             `json:"id"`
	CategoryID  int64             `json:"category_id"`
	Slug        string            `json:"slug"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ImageURL    string            `json:"image_url"`
	CreatedAt   time.Time         `json:"created_at"`
	Variants    []variantResponse `json:"variants"`
}

// paginated is a generic envelope for list endpoints — the same shape for
// products now, orders later.
type paginated[T any] struct {
	Items   []T `json:"items"`
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

func toProductResponse(p domain.Product) productResponse {
	variants := make([]variantResponse, 0, len(p.Variants))
	for _, v := range p.Variants {
		variants = append(variants, variantResponse{
			ID: v.ID, SKU: v.SKU, Label: v.Label,
			PriceMinor: v.PriceMinor, StockQty: v.StockQty,
		})
	}
	return productResponse{
		ID: p.ID, CategoryID: p.CategoryID, Slug: p.Slug, Name: p.Name,
		Description: p.Description, ImageURL: p.ImageURL,
		CreatedAt: p.CreatedAt, Variants: variants,
	}
}

func (s *Server) handleListProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := domain.ProductFilter{
		CategorySlug: q.Get("category"),
		Search:       q.Get("q"),
		Page:         intQueryParam(q.Get("page"), 1),
		PerPage:      intQueryParam(q.Get("per_page"), 20),
	}
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
		items = append(items, toProductResponse(p))
	}
	s.respondJSON(w, http.StatusOK, paginated[productResponse]{
		Items: items, Page: filter.Page, PerPage: filter.PerPage, Total: total,
	})
}

func (s *Server) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	p, err := s.store.GetProductBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.respondError(w, http.StatusNotFound, "not_found", "no such product")
			return
		}
		s.log.Error("getting product", "slug", slug, "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}
	s.respondJSON(w, http.StatusOK, toProductResponse(p))
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
