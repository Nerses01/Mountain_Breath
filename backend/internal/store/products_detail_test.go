package store_test

import (
	"context"
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

func productIDBySlug(t *testing.T, slug string) int64 {
	t.Helper()
	var id int64
	if err := testPool.QueryRow(context.Background(),
		`SELECT id FROM products WHERE slug = $1`, slug).Scan(&id); err != nil {
		t.Fatalf("looking up %q: %v", slug, err)
	}
	return id
}

func imageState(t *testing.T, productID int64) []struct {
	ID        int64
	URL       string
	SortOrder int
	IsPrimary bool
} {
	t.Helper()
	rows, err := testPool.Query(context.Background(), `
		SELECT id, url, sort_order, is_primary
		FROM product_images WHERE product_id = $1
		ORDER BY sort_order, id`, productID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var out []struct {
		ID        int64
		URL       string
		SortOrder int
		IsPrimary bool
	}
	for rows.Next() {
		var r struct {
			ID        int64
			URL       string
			SortOrder int
			IsPrimary bool
		}
		if err := rows.Scan(&r.ID, &r.URL, &r.SortOrder, &r.IsPrimary); err != nil {
			t.Fatal(err)
		}
		out = append(out, r)
	}
	return out
}

// The partial unique index in 000011 is the whole point of the images table's
// design, so it gets tested directly rather than trusted.
func TestProductImages_OnePrimaryPerProduct(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := productIDBySlug(t, "honey")
	wax := productIDBySlug(t, "wax")

	// The FIRST image becomes the hero without being asked to — a gallery
	// with no primary renders no hero at all.
	first, err := s.AddProductImage(ctx, honey, "/uploads/a.jpg", map[domain.Locale]string{
		domain.LocaleEN: "A jar of honey", domain.LocaleHY: "Մեղրի բանկա",
	})
	if err != nil {
		t.Fatalf("AddProductImage: %v", err)
	}
	if !first.IsPrimary {
		t.Error("the first image of a product should become its primary")
	}

	second, err := s.AddProductImage(ctx, honey, "/uploads/b.jpg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if second.IsPrimary {
		t.Error("a second image must not also be primary")
	}
	if second.SortOrder <= first.SortOrder {
		t.Errorf("sort_order = %d, want after the first (%d)", second.SortOrder, first.SortOrder)
	}

	// A DIFFERENT product may of course have its own primary — this is what
	// a naive UNIQUE (product_id, is_primary) would still allow, but a naive
	// UNIQUE (is_primary) would not.
	if _, err := s.AddProductImage(ctx, wax, "/uploads/c.jpg", nil); err != nil {
		t.Fatalf("second product's first image: %v", err)
	}

	// Writing the flag directly, bypassing the store, must be rejected by
	// the DATABASE — that is the difference between a constraint and a
	// convention.
	_, err = testPool.Exec(ctx,
		`UPDATE product_images SET is_primary = TRUE WHERE id = $1`, second.ID)
	if err == nil {
		t.Fatal("a second primary image was accepted; the partial unique index is not doing its job")
	}
	if !strings.Contains(err.Error(), "idx_product_images_one_primary") {
		t.Errorf("rejected by the wrong constraint: %v", err)
	}

	// ...while the store's own path succeeds, because it clears the old flag
	// first inside the transaction.
	if err := s.SaveProductImages(ctx, honey, []domain.ProductImage{
		{ID: second.ID, SortOrder: 0, IsPrimary: true},
		{ID: first.ID, SortOrder: 1, IsPrimary: false},
	}, nil); err != nil {
		t.Fatalf("SaveProductImages: %v", err)
	}
	imgs := imageState(t, honey)
	if len(imgs) != 2 || imgs[0].ID != second.ID || !imgs[0].IsPrimary || imgs[1].IsPrimary {
		t.Errorf("after reorder: %+v", imgs)
	}
}

func TestProductImages_DeletingThePrimaryPromotesAnother(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := productIDBySlug(t, "honey")
	first, _ := s.AddProductImage(ctx, honey, "/uploads/a.jpg", nil)
	second, _ := s.AddProductImage(ctx, honey, "/uploads/b.jpg", nil)

	if err := s.DeleteProductImage(ctx, honey, first.ID); err != nil {
		t.Fatalf("DeleteProductImage: %v", err)
	}

	imgs := imageState(t, honey)
	if len(imgs) != 1 || imgs[0].ID != second.ID {
		t.Fatalf("images after delete: %+v", imgs)
	}
	// Otherwise the product silently loses its hero and the page renders a
	// gallery with nothing at the top.
	if !imgs[0].IsPrimary {
		t.Error("deleting the primary image left the product with no primary")
	}

	// Deleting the last one is legitimate and must not try to promote a row
	// that is not there.
	if err := s.DeleteProductImage(ctx, honey, second.ID); err != nil {
		t.Fatalf("deleting the last image: %v", err)
	}
	if imgs := imageState(t, honey); len(imgs) != 0 {
		t.Errorf("expected an empty gallery, got %+v", imgs)
	}

	// Another product's id must not be usable to delete this one's images.
	third, _ := s.AddProductImage(ctx, honey, "/uploads/d.jpg", nil)
	wax := productIDBySlug(t, "wax")
	if err := s.DeleteProductImage(ctx, wax, third.ID); err == nil {
		t.Error("deleted an image through the wrong product id")
	}
}

func TestProductImages_AltFollowsLocale(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := productIDBySlug(t, "honey")
	if _, err := s.AddProductImage(ctx, honey, "/uploads/a.jpg", map[domain.Locale]string{
		domain.LocaleEN: "A jar of honey",
		domain.LocaleHY: "Մեղրի բանկա",
	}); err != nil {
		t.Fatal(err)
	}

	hy, err := s.GetProductBySlug(ctx, "honey", domain.View{Locale: domain.LocaleHY})
	if err != nil {
		t.Fatal(err)
	}
	if len(hy.Images) != 1 || hy.Images[0].Alt != "Մեղրի բանկա" {
		t.Errorf("Armenian alt = %+v", hy.Images)
	}

	// Russian was never written, so it falls back to English rather than
	// leaving a screen reader with nothing to announce.
	ru, err := s.GetProductBySlug(ctx, "honey", domain.View{Locale: domain.LocaleRU})
	if err != nil {
		t.Fatal(err)
	}
	if ru.Images[0].Alt != "A jar of honey" {
		t.Errorf("Russian alt = %q, want the English fallback", ru.Images[0].Alt)
	}
}

// Editorial content is replaced per locale, and the fallback happens per
// LIST rather than per row — the two properties decision #4 bought.
func TestProductEditorial_ReplacePerLocaleAndListFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := productIDBySlug(t, "honey")

	if err := s.SaveProductEditorial(ctx, honey, map[domain.Locale]domain.ProductEditorial{
		domain.LocaleEN: {
			Highlights: []domain.ProductHighlight{{Text: "First"}, {Text: "Second"}, {Text: "Third"}},
			UsageCards: []domain.ProductUsageCard{{Kicker: "Morning", Title: "A spoon", Body: "Plain."}},
		},
		domain.LocaleHY: {
			// Deliberately a DIFFERENT number of bullets — the shape decision
			// #4 chose is what allows a translator not to pad.
			Highlights: []domain.ProductHighlight{{Text: "Առաջին"}, {Text: "Երկրորդ"}},
		},
	}); err != nil {
		t.Fatalf("SaveProductEditorial: %v", err)
	}

	en, err := s.GetProductBySlug(ctx, "honey", domain.View{})
	if err != nil {
		t.Fatal(err)
	}
	if len(en.Highlights) != 3 || en.Highlights[0].Text != "First" {
		t.Errorf("English highlights = %+v", en.Highlights)
	}

	hy, err := s.GetProductBySlug(ctx, "honey", domain.View{Locale: domain.LocaleHY})
	if err != nil {
		t.Fatal(err)
	}
	if len(hy.Highlights) != 2 {
		t.Errorf("Armenian highlights = %d, want 2 — languages may differ in count", len(hy.Highlights))
	}
	// Armenian usage cards were never written, so the WHOLE English list
	// shows. Falling back per row would have interleaved two languages
	// inside one panel.
	if len(hy.UsageCards) != 1 || hy.UsageCards[0].Kicker != "Morning" {
		t.Errorf("Armenian usage cards = %+v, want the English list", hy.UsageCards)
	}

	// A shorter list must REPLACE, not merge: the PK includes sort_order, so
	// an upsert would strand the third bullet forever.
	if err := s.SaveProductEditorial(ctx, honey, map[domain.Locale]domain.ProductEditorial{
		domain.LocaleEN: {Highlights: []domain.ProductHighlight{{Text: "Only one now"}}},
	}); err != nil {
		t.Fatal(err)
	}
	en, err = s.GetProductBySlug(ctx, "honey", domain.View{})
	if err != nil {
		t.Fatal(err)
	}
	if len(en.Highlights) != 1 {
		t.Errorf("after shortening: %+v — replace, not merge", en.Highlights)
	}
	// ...and the untouched locale is untouched.
	hy, _ = s.GetProductBySlug(ctx, "honey", domain.View{Locale: domain.LocaleHY})
	if len(hy.Highlights) != 2 {
		t.Errorf("editing English changed the Armenian list: %+v", hy.Highlights)
	}
}

func TestListRelated_CuratedBeatsFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	jelly := productIDBySlug(t, "jelly")
	wax := productIDBySlug(t, "wax")

	// With nothing curated, the fallback ranks by SHARED BENEFITS then
	// popularity. jelly is energy+skin; honey shares energy (sales 100),
	// pollen shares energy (70), wax shares skin (10).
	got, err := s.ListRelated(ctx, "jelly", domain.View{})
	if err != nil {
		t.Fatalf("ListRelated: %v", err)
	}
	if slugs := slugsOf(got); !equalSlugs(slugs, "honey", "pollen", "wax") {
		t.Errorf("fallback order = %v, want honey, pollen, wax", slugs)
	}

	// The product itself must never appear in its own panel.
	for _, p := range got {
		if p.Slug == "jelly" {
			t.Error("a product was listed as related to itself")
		}
	}

	// Curating anything replaces the computed list entirely, in the admin's
	// order — not merged with it, or the curation would be a suggestion.
	if err := s.SaveProductRelated(ctx, jelly, []int64{wax}); err != nil {
		t.Fatalf("SaveProductRelated: %v", err)
	}
	got, err = s.ListRelated(ctx, "jelly", domain.View{})
	if err != nil {
		t.Fatal(err)
	}
	if slugs := slugsOf(got); !equalSlugs(slugs, "wax") {
		t.Errorf("curated = %v, want just wax", slugs)
	}

	// The cards in the panel need prices and benefits, like any other card.
	if len(got[0].Variants) == 0 || len(got[0].Benefits) == 0 {
		t.Errorf("related card is missing variants or benefits: %+v", got[0])
	}
}

func TestSaveProductRelated_DropsSelfReferences(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	jelly := productIDBySlug(t, "jelly")
	honey := productIDBySlug(t, "honey")

	// A shop owner ticking the product they are editing is an obvious slip;
	// answering it with a constraint violation would be worse than quietly
	// doing the sensible thing. Duplicates go the same way.
	if err := s.SaveProductRelated(ctx, jelly, []int64{jelly, honey, honey}); err != nil {
		t.Fatalf("SaveProductRelated: %v", err)
	}

	got, err := s.ListRelated(ctx, "jelly", domain.View{})
	if err != nil {
		t.Fatal(err)
	}
	if slugs := slugsOf(got); !equalSlugs(slugs, "honey") {
		t.Errorf("related = %v, want just honey", slugs)
	}
}

// The listing must NOT pay for editorial content — the split is the reason
// the two endpoints exist.
func TestListProducts_LeavesEditorialEmpty(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := productIDBySlug(t, "honey")
	if _, err := s.AddProductImage(ctx, honey, "/uploads/a.jpg", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProductEditorial(ctx, honey, map[domain.Locale]domain.ProductEditorial{
		domain.LocaleEN: {Highlights: []domain.ProductHighlight{{Text: "A bullet"}}},
	}); err != nil {
		t.Fatal(err)
	}

	list, _, err := s.ListProducts(ctx, domain.ProductFilter{
		Page: 1, PerPage: 20, CategorySlug: "honey",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("got %d products", len(list))
	}
	if len(list[0].Images) != 0 || len(list[0].Highlights) != 0 || len(list[0].UsageCards) != 0 {
		t.Errorf("the listing loaded editorial content it does not render: %+v", list[0])
	}

	// ...while the detail read does load it.
	detail, err := s.GetProductBySlug(ctx, "honey", domain.View{})
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Images) != 1 || len(detail.Highlights) != 1 {
		t.Errorf("detail is missing editorial content: %+v", detail)
	}
}
