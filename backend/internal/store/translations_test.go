package store_test

import (
	"context"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// seedTranslatable creates one category and one product, with English text
// on the parent rows plus translation rows, and returns the store.
func seedTranslatable(t *testing.T) *store.Store {
	t.Helper()
	resetDB(t)
	ctx := context.Background()

	_, err := testPool.Exec(ctx, `
		INSERT INTO categories (slug, name, sort_order) VALUES ('honey', 'Honey', 1);

		INSERT INTO products (category_id, slug, name, description)
		SELECT id, 'wildflower-honey', 'Wildflower Honey', 'Raw honey from alpine fields'
		FROM categories WHERE slug = 'honey';

		WITH variant AS (
			INSERT INTO product_variants (product_id, sku, label, stock_qty)
			SELECT id, 'HON-1', '350 g', 10 FROM products WHERE slug = 'wildflower-honey'
			RETURNING id
		)
		INSERT INTO variant_prices (variant_id, currency, price_minor)
		SELECT id, 'USD', 520000 FROM variant;

		-- English translations, as migration 000007's backfill would create.
		INSERT INTO category_translations (category_id, locale, name)
		SELECT id, 'en', 'Honey' FROM categories WHERE slug = 'honey';
		INSERT INTO product_translations (product_id, locale, name, description)
		SELECT id, 'en', 'Wildflower Honey', 'Raw honey from alpine fields'
		FROM products WHERE slug = 'wildflower-honey';

		-- Russian, but deliberately NO Armenian: the gap is the point.
		INSERT INTO category_translations (category_id, locale, name)
		SELECT id, 'ru', 'Мёд' FROM categories WHERE slug = 'honey';
		INSERT INTO product_translations (product_id, locale, name, description)
		SELECT id, 'ru', 'Цветочный мёд', 'Сырой мёд с альпийских лугов'
		FROM products WHERE slug = 'wildflower-honey';
	`)
	if err != nil {
		t.Fatalf("seeding: %v", err)
	}
	return store.New(testPool)
}

func TestListCategoriesTranslates(t *testing.T) {
	if testing.Short() {
		t.Skip("needs Docker")
	}
	s := seedTranslatable(t)
	ctx := context.Background()

	t.Run("serves the requested language", func(t *testing.T) {
		cats, err := s.ListCategories(ctx, domain.LocaleRU)
		if err != nil {
			t.Fatalf("listing: %v", err)
		}
		if cats[0].Name != "Мёд" {
			t.Errorf("name = %q; want Мёд", cats[0].Name)
		}
	})

	t.Run("falls back to English when a translation is missing", func(t *testing.T) {
		// No Armenian row exists. A missing translation must degrade to
		// readable English, never to an empty string.
		cats, err := s.ListCategories(ctx, domain.LocaleHY)
		if err != nil {
			t.Fatalf("listing: %v", err)
		}
		if cats[0].Name != "Honey" {
			t.Errorf("name = %q; want the English fallback Honey", cats[0].Name)
		}
	})
}

func TestGetProductBySlugTranslates(t *testing.T) {
	if testing.Short() {
		t.Skip("needs Docker")
	}
	s := seedTranslatable(t)
	ctx := context.Background()

	ru, err := s.GetProductBySlug(ctx, "wildflower-honey", domain.View{Locale: domain.LocaleRU})
	if err != nil {
		t.Fatalf("getting product: %v", err)
	}
	if ru.Name != "Цветочный мёд" {
		t.Errorf("ru name = %q; want Цветочный мёд", ru.Name)
	}
	// The slug is identity, not display text: the same URL must resolve in
	// every language so a shared link works between speakers.
	if ru.Slug != "wildflower-honey" {
		t.Errorf("slug changed with locale: %q", ru.Slug)
	}

	hy, err := s.GetProductBySlug(ctx, "wildflower-honey", domain.View{Locale: domain.LocaleHY})
	if err != nil {
		t.Fatalf("getting product: %v", err)
	}
	if hy.Name != "Wildflower Honey" {
		t.Errorf("hy name = %q; want the English fallback", hy.Name)
	}
}

func TestSearchUsesPerLocaleStemming(t *testing.T) {
	if testing.Short() {
		t.Skip("needs Docker")
	}
	s := seedTranslatable(t)
	ctx := context.Background()

	find := func(locale domain.Locale, query string) int {
		t.Helper()
		_, total, err := s.ListProducts(ctx, domain.ProductFilter{
			Search: query, Page: 1, PerPage: 10, Locale: locale,
		})
		if err != nil {
			t.Fatalf("searching %q in %s: %v", query, locale, err)
		}
		return total
	}

	// Russian stemming: "мёд" is stored, "мед" is searched. The russian
	// configuration normalises ё to е, so this only matches because the
	// query was stemmed with the RIGHT configuration — under the english
	// config it would not.
	if got := find(domain.LocaleRU, "мед"); got != 1 {
		t.Errorf("ru search for мед found %d; want 1", got)
	}

	// English stemming still works in its own locale: "fields" is stored,
	// "field" is searched.
	if got := find(domain.LocaleEN, "field"); got != 1 {
		t.Errorf("en search for field found %d; want 1", got)
	}

	// A product with no Armenian translation is still reachable in Armenian
	// through the language-agnostic trigram door, using its English text.
	if got := find(domain.LocaleHY, "wildflower"); got != 1 {
		t.Errorf("hy search for wildflower found %d; want 1 (trigram fallback)", got)
	}
}
