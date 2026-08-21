package store_test

import (
	"context"
	"errors"
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

// The photo cap is a count the database cannot express declaratively, so it
// lives in AddProductImage's transaction — and gets tested like a constraint
// would be: by pushing past it.
func TestProductImages_CapAtThree(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := productIDBySlug(t, "honey")

	for i := 0; i < domain.MaxGalleryImages; i++ {
		if _, err := s.AddProductImage(ctx, honey, "/uploads/a.jpg", nil); err != nil {
			t.Fatalf("image %d within the cap: %v", i+1, err)
		}
	}
	if _, err := s.AddProductImage(ctx, honey, "/uploads/d.jpg", nil); !errors.Is(err, domain.ErrGalleryFull) {
		t.Errorf("4th image: got %v, want ErrGalleryFull", err)
	}

	// The video occupies its own slot — it must not eat a photo's place.
	wax := productIDBySlug(t, "wax")
	if _, err := s.AddProductVideo(ctx, wax, "/uploads/w.mp4"); err != nil {
		t.Fatalf("AddProductVideo: %v", err)
	}
	for i := 0; i < domain.MaxGalleryImages; i++ {
		if _, err := s.AddProductImage(ctx, wax, "/uploads/b.jpg", nil); err != nil {
			t.Fatalf("image %d beside a video: %v", i+1, err)
		}
	}

	// Deleting a photo reopens the slot — the cap counts rows, not history.
	imgs := imageState(t, honey)
	if err := s.DeleteProductImage(ctx, honey, imgs[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddProductImage(ctx, honey, "/uploads/e.jpg", nil); err != nil {
		t.Errorf("re-adding after a delete: %v", err)
	}
}

func TestProductVideo_SinglePerProduct(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := productIDBySlug(t, "honey")

	video, err := s.AddProductVideo(ctx, honey, "/uploads/v.mp4")
	if err != nil {
		t.Fatalf("AddProductVideo: %v", err)
	}
	if _, err := s.AddProductVideo(ctx, honey, "/uploads/v2.mp4"); !errors.Is(err, domain.ErrVideoExists) {
		t.Errorf("second video: got %v, want ErrVideoExists", err)
	}
	// Replacing means delete-then-upload, through the same DELETE route
	// photos use — the video is an image row on purpose.
	if err := s.DeleteProductImage(ctx, honey, video.ID); err != nil {
		t.Fatalf("deleting the video: %v", err)
	}
	if _, err := s.AddProductVideo(ctx, honey, "/uploads/v3.mp4"); err != nil {
		t.Errorf("re-adding after delete: %v", err)
	}

	// A missing product is a 404, not a 500.
	if _, err := s.AddProductVideo(ctx, 99999, "/uploads/v.mp4"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown product: got %v, want ErrNotFound", err)
	}

	// The detail read hands the video to its own field: Images feeds <img>
	// slots, Video a <video> tab, and mixing them would put an .mp4 in an
	// <img> on every consumer that renders the array blindly.
	if _, err := s.AddProductImage(ctx, honey, "/uploads/a.jpg", nil); err != nil {
		t.Fatal(err)
	}
	detail, err := s.GetProductBySlug(ctx, "honey", domain.View{})
	if err != nil {
		t.Fatal(err)
	}
	if detail.Video == nil || detail.Video.URL != "/uploads/v3.mp4" {
		t.Errorf("Video = %+v, want the uploaded clip", detail.Video)
	}
	if len(detail.Images) != 1 || detail.Images[0].URL != "/uploads/a.jpg" {
		t.Errorf("Images = %+v — the video must not appear among the photos", detail.Images)
	}
}

// The video must never become the hero: not on upload (constraint), not by
// promotion when the last photo-hero is deleted (the store's WHERE), and a
// product whose first PHOTO arrives after the video still auto-heroes it.
func TestProductVideo_NeverPromotedToHero(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	seedHive(t)
	s := store.New(testPool)
	ctx := context.Background()

	honey := productIDBySlug(t, "honey")

	if _, err := s.AddProductVideo(ctx, honey, "/uploads/v.mp4"); err != nil {
		t.Fatal(err)
	}
	// The video is already there; the first photo must still become primary
	// (the auto-hero count filters on kind).
	photo, err := s.AddProductImage(ctx, honey, "/uploads/a.jpg", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !photo.IsPrimary {
		t.Error("first photo beside an existing video should still auto-hero")
	}

	// Deleting the only photo: the promotion must SKIP the video and leave
	// the product hero-less rather than 500 on the video-not-primary CHECK.
	if err := s.DeleteProductImage(ctx, honey, photo.ID); err != nil {
		t.Fatalf("deleting the hero beside a video: %v", err)
	}
	var primaries int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM product_images WHERE product_id = $1 AND is_primary`,
		honey).Scan(&primaries); err != nil {
		t.Fatal(err)
	}
	if primaries != 0 {
		t.Errorf("got %d primary rows, want 0 — the video must not be promoted", primaries)
	}

	// Belt and braces: the CHECK itself rejects a direct flag write.
	if _, err := testPool.Exec(ctx, `
		UPDATE product_images SET is_primary = TRUE
		WHERE product_id = $1 AND kind = 'video'`, honey); err == nil {
		t.Error("the database accepted a primary video; the CHECK is not doing its job")
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
// the two endpoints exist. Photos are the exception since decision #99: the
// card's hover slideshow renders every photo, so the list attaches them
// (thin: url+alt), while bullets and usage cards stay detail-only.
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
	if len(list[0].Highlights) != 0 || len(list[0].UsageCards) != 0 {
		t.Errorf("the listing loaded editorial content it does not render: %+v", list[0])
	}
	if len(list[0].Images) != 1 || list[0].Images[0].URL != "/uploads/a.jpg" {
		t.Errorf("the listing should carry the card's photos: %+v", list[0].Images)
	}
	if list[0].Video != nil {
		t.Error("the listing loaded the video, which no card plays")
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
