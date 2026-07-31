package store_test

import (
	"context"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

func TestProductSearch(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	// Three products: "honey" in the NAME, "honey" in the DESCRIPTION only,
	// and one that never mentions honey at all.
	if _, err := testPool.Exec(ctx, `
		WITH cat AS (INSERT INTO categories (slug, name) VALUES ('food', 'Food') RETURNING id)
		INSERT INTO products (category_id, slug, name, description)
		SELECT id, v.slug, v.name, v.description FROM cat, (VALUES
			('wildflower-honey', 'Wildflower Honey', 'Raw and golden.'),
			('herbal-tea', 'Herbal Tea', 'Delicious with honey and lemon.'),
			('ground-coffee', 'Ground Coffee', 'Dark roast for the cezve.')
		) AS v(slug, name, description)`); err != nil {
		t.Fatal(err)
	}

	search := func(q string) []domain.Product {
		t.Helper()
		products, _, err := s.ListProducts(ctx, domain.ProductFilter{
			Search: q, Page: 1, PerPage: 10,
		})
		if err != nil {
			t.Fatalf("search %q: %v", q, err)
		}
		return products
	}

	t.Run("matches name and description, ranks name first", func(t *testing.T) {
		got := search("honey")
		if len(got) != 2 {
			t.Fatalf("got %d results, want 2", len(got))
		}
		// setweight A (name) must outrank weight B (description)
		if got[0].Slug != "wildflower-honey" || got[1].Slug != "herbal-tea" {
			t.Errorf("ranking wrong: got [%s, %s]", got[0].Slug, got[1].Slug)
		}
	})

	t.Run("stemming: singular query finds plural-ish forms", func(t *testing.T) {
		// 'english' config stems: "roasted"/"roast" share a lexeme
		if got := search("roasted"); len(got) != 1 || got[0].Slug != "ground-coffee" {
			t.Errorf("stemmed search failed: %+v", got)
		}
	})

	t.Run("no match means empty result, not error", func(t *testing.T) {
		if got := search("spaceship"); len(got) != 0 {
			t.Errorf("got %d results for nonsense query", len(got))
		}
	})

	t.Run("websearch syntax: exclusion with minus", func(t *testing.T) {
		got := search("honey -tea")
		if len(got) != 1 || got[0].Slug != "wildflower-honey" {
			t.Errorf("exclusion failed: %+v", got)
		}
	})

	t.Run("search count matches filtered total", func(t *testing.T) {
		_, total, err := s.ListProducts(ctx, domain.ProductFilter{Search: "honey", Page: 1, PerPage: 1})
		if err != nil {
			t.Fatal(err)
		}
		if total != 2 {
			t.Errorf("total = %d, want 2", total)
		}
	})
}
