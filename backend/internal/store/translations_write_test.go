package store_test

import (
	"context"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// translationRow reads one locale's stored text directly, bypassing the
// read-path fallback — otherwise a missing row would be indistinguishable
// from a correctly written English one.
func translationRow(t *testing.T, table, fk string, id int64, locale domain.Locale) (string, bool) {
	t.Helper()
	var name string
	err := testPool.QueryRow(context.Background(),
		"SELECT name FROM "+table+" WHERE "+fk+" = $1 AND locale = $2", id, locale,
	).Scan(&name)
	if err != nil {
		return "", false
	}
	return name, true
}

func TestCreateCategoryWritesTranslations(t *testing.T) {
	if testing.Short() {
		t.Skip("needs Docker")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	cat := domain.Category{
		Slug: "honey", Name: "Honey", SortOrder: 1,
		Translations: map[domain.Locale]string{
			domain.LocaleHY: "Մեղր",
			domain.LocaleRU: "Мёд",
		},
	}
	if err := s.CreateCategory(ctx, &cat); err != nil {
		t.Fatalf("creating category: %v", err)
	}

	// English is written from Name, never from Translations — that is why the
	// API rejects an "en" key.
	for locale, want := range map[domain.Locale]string{
		domain.LocaleEN: "Honey",
		domain.LocaleHY: "Մեղր",
		domain.LocaleRU: "Мёд",
	} {
		got, ok := translationRow(t, "category_translations", "category_id", cat.ID, locale)
		if !ok {
			t.Errorf("no %s translation row written", locale)
			continue
		}
		if got != want {
			t.Errorf("%s name = %q; want %q", locale, got, want)
		}
	}
}

func TestCreateCategoryWritesEnglishOnlyWhenNoTranslations(t *testing.T) {
	if testing.Short() {
		t.Skip("needs Docker")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	cat := domain.Category{Slug: "beeswax", Name: "Beeswax", SortOrder: 2}
	if err := s.CreateCategory(ctx, &cat); err != nil {
		t.Fatalf("creating category: %v", err)
	}

	if _, ok := translationRow(t, "category_translations", "category_id", cat.ID, domain.LocaleEN); !ok {
		t.Error("English translation row must always be written")
	}
	// Omitting a language must not fabricate an empty row — the read path
	// falls back to English, and a blank row would defeat that.
	if got, ok := translationRow(t, "category_translations", "category_id", cat.ID, domain.LocaleHY); ok {
		t.Errorf("unexpected hy row with value %q", got)
	}
}

func TestCreateCategoryRollsBackOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("needs Docker")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	first := domain.Category{Slug: "honey", Name: "Honey"}
	if err := s.CreateCategory(ctx, &first); err != nil {
		t.Fatalf("seeding: %v", err)
	}

	// Same slug: the INSERT fails, and the transaction must leave nothing
	// behind — no orphaned translation rows from the second attempt.
	dup := domain.Category{
		Slug: "honey", Name: "Honey again",
		Translations: map[domain.Locale]string{domain.LocaleRU: "Мёд"},
	}
	if err := s.CreateCategory(ctx, &dup); err != domain.ErrSlugTaken {
		t.Fatalf("err = %v; want ErrSlugTaken", err)
	}

	var count int
	if err := testPool.QueryRow(ctx,
		"SELECT count(*) FROM category_translations WHERE locale = 'ru'").Scan(&count); err != nil {
		t.Fatalf("counting: %v", err)
	}
	if count != 0 {
		t.Errorf("%d Russian rows survived a rolled-back create", count)
	}
}

func TestProductTranslationsRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("needs Docker")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	cat := domain.Category{Slug: "honey", Name: "Honey"}
	if err := s.CreateCategory(ctx, &cat); err != nil {
		t.Fatalf("seeding category: %v", err)
	}

	p := domain.Product{
		CategoryID: cat.ID, Slug: "wildflower-honey",
		Name: "Wildflower Honey", Description: "Raw honey", IsActive: true,
		Variants: []domain.ProductVariant{
			{SKU: "HON-1", Label: "350 g", PriceMinor: 520000, StockQty: 5},
		},
		Translations: map[domain.Locale]domain.ProductText{
			domain.LocaleRU: {Name: "Цветочный мёд", Description: "Сырой мёд"},
		},
	}
	if err := s.CreateProduct(ctx, &p); err != nil {
		t.Fatalf("creating product: %v", err)
	}

	// Written text must come back through the READ path in the same language.
	ru, err := s.GetProductBySlug(ctx, "wildflower-honey", domain.LocaleRU)
	if err != nil {
		t.Fatalf("reading ru: %v", err)
	}
	if ru.Name != "Цветочный мёд" {
		t.Errorf("ru name = %q; want Цветочный мёд", ru.Name)
	}

	// An untranslated language still falls back rather than blanking.
	hy, err := s.GetProductBySlug(ctx, "wildflower-honey", domain.LocaleHY)
	if err != nil {
		t.Fatalf("reading hy: %v", err)
	}
	if hy.Name != "Wildflower Honey" {
		t.Errorf("hy name = %q; want the English fallback", hy.Name)
	}
}

func TestUpdateProductUpsertsTranslations(t *testing.T) {
	if testing.Short() {
		t.Skip("needs Docker")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	cat := domain.Category{Slug: "honey", Name: "Honey"}
	if err := s.CreateCategory(ctx, &cat); err != nil {
		t.Fatalf("seeding category: %v", err)
	}
	p := domain.Product{
		CategoryID: cat.ID, Slug: "honey-jar", Name: "Honey Jar", IsActive: true,
		Variants: []domain.ProductVariant{
			{SKU: "HJ-1", Label: "350 g", PriceMinor: 100, StockQty: 1},
		},
		Translations: map[domain.Locale]domain.ProductText{
			domain.LocaleRU: {Name: "Банка мёда"},
		},
	}
	if err := s.CreateProduct(ctx, &p); err != nil {
		t.Fatalf("creating: %v", err)
	}

	// Update: change the Russian text and add Armenian. The upsert must
	// replace the existing row rather than fail on the primary key.
	p.Name = "Honey Jar (new)"
	p.Translations = map[domain.Locale]domain.ProductText{
		domain.LocaleRU: {Name: "Банка мёда, обновлено"},
		domain.LocaleHY: {Name: "Մեղրի բանկա"},
	}
	if err := s.UpdateProduct(ctx, &p); err != nil {
		t.Fatalf("updating: %v", err)
	}

	for locale, want := range map[domain.Locale]string{
		domain.LocaleEN: "Honey Jar (new)",
		domain.LocaleRU: "Банка мёда, обновлено",
		domain.LocaleHY: "Մեղրի բանկա",
	} {
		got, ok := translationRow(t, "product_translations", "product_id", p.ID, locale)
		if !ok {
			t.Errorf("no %s row after update", locale)
			continue
		}
		if got != want {
			t.Errorf("%s name = %q; want %q", locale, got, want)
		}
	}
}
