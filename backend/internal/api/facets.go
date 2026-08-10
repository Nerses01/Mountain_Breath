package api

import (
	"net/http"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// GET /catalog/facets — everything the Shop sidebar needs to draw itself.
//
// A SEPARATE endpoint from GET /products, though both read the same filter,
// because the two answer different questions and change at different rates.
// The grid changes with the page number; the sidebar does not. Keeping them
// apart lets the client cache them under different query keys, so paging
// through results never re-runs the expensive counting query.

type facetCountResponse struct {
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type facetsResponse struct {
	Categories []facetCountResponse `json:"categories"`
	Benefits   []facetCountResponse `json:"benefits"`
	Total      int                  `json:"total"`
	// Bounds of the price slider, in minor units like every other money
	// value in this API. Both zero means nothing matched the filter.
	PriceMinMinor int64 `json:"price_min_minor"`
	PriceMaxMinor int64 `json:"price_max_minor"`
}

func toFacetCounts(in []domain.FacetCount) []facetCountResponse {
	out := make([]facetCountResponse, 0, len(in))
	for _, f := range in {
		out = append(out, facetCountResponse{Slug: f.Slug, Name: f.Name, Count: f.Count})
	}
	return out
}

func (s *Server) handleCatalogFacets(w http.ResponseWriter, r *http.Request) {
	// Page and PerPage stay at their zero values: facets count the WHOLE
	// filtered catalog, not the current page. That is the entire point of
	// the endpoint — a sidebar that only counted the visible rows would say
	// "Honey 3" on page one and "Honey 1" on page two.
	facets, err := s.store.CatalogFacets(r.Context(), productFilterFromQuery(r))
	if err != nil {
		s.log.Error("listing catalog facets", "error", err)
		s.respondError(w, http.StatusInternalServerError, "internal_error", "internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, facetsResponse{
		Categories:    toFacetCounts(facets.Categories),
		Benefits:      toFacetCounts(facets.Benefits),
		Total:         facets.Total,
		PriceMinMinor: facets.PriceMinMinor,
		PriceMaxMinor: facets.PriceMaxMinor,
	})
}
