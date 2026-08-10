package domain

// Benefit is one chip of the "Good for" taxonomy — Energy, Immunity, Skin,
// Recovery, Sweetening.
//
// It looks like Category and is shaped differently on purpose: a product has
// exactly one category and any number of benefits (migration 000008). Name
// follows the same read/write duality as Category.Name — English on a write,
// the resolved text for the requested locale on a read.
type Benefit struct {
	ID        int64
	Slug      string
	SortOrder int
	Name      string
}

// FacetCount is one row of a filter sidebar: what to show, what to put in
// the URL, and how many products would remain if it were clicked.
type FacetCount struct {
	Slug  string
	Name  string
	Count int
}

// CatalogFacets is everything the Shop sidebar needs to render itself,
// answered in one round trip.
//
// The counts are the expensive part, and the reason this is its own endpoint
// rather than extra fields on GET /products: they cannot be derived from the
// page of products the grid shows. "Honey 1" has to be true across the whole
// filtered catalog, not across the 12 rows on screen — so the database
// counts every matching product for every facet value, on every filter
// change. That is the standing cost of faceted search, and why real shops
// cache these aggressively.
type CatalogFacets struct {
	// Categories and Benefits list EVERY value, including zero counts, so
	// the sidebar keeps its shape as filters narrow instead of items
	// disappearing under the cursor.
	Categories []FacetCount
	Benefits   []FacetCount

	// Total is the "All hive products" row: the count with the category
	// filter lifted but the other filters applied.
	Total int

	// Price bounds across the matching products, in minor units — the ends
	// of the sidebar's dual slider. Zero when nothing matches.
	PriceMinMinor int64
	PriceMaxMinor int64
}
