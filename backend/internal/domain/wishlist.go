package domain

import "time"

// WishlistItem is a saved product plus WHEN it was saved (A3) — the canvas
// card's "saved 2 weeks ago" line. Embedding keeps every Product consumer
// working on the item unchanged; SavedAt rides along.
//
// The C++ analogue is public inheritance used purely for is-a convenience —
// except Go embedding is composition that FORWARDS (promoted fields), with
// no virtual dispatch to reason about.
type WishlistItem struct {
	Product
	SavedAt time.Time
}
