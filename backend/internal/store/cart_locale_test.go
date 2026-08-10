package store_test

import (
	"context"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// The cart shows product names, so it has to speak the same language as the
// page around it. It did not until E2: E1.5 translated the catalog and left
// GetCart reading products.name, which fails SILENTLY — the fallback returns
// perfectly valid English, so nothing errors and nobody notices until an
// Armenian customer looks at their basket.
func TestGetCart_ResolvesNamesInTheRequestedLocale(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	userID := seedUserWithCart(t, "shopper@test.local", variantID, 2)

	if _, err := testPool.Exec(ctx, `
		INSERT INTO product_translations (product_id, locale, name, description)
		SELECT v.product_id, 'hy', 'Վայրի մեղր', ''
		FROM product_variants v WHERE v.id = $1`, variantID); err != nil {
		t.Fatal(err)
	}

	hy, err := s.GetCart(ctx, userID, domain.LocaleHY)
	if err != nil {
		t.Fatalf("GetCart: %v", err)
	}
	if len(hy) != 1 || hy[0].ProductName != "Վայրի մեղր" {
		t.Errorf("Armenian cart = %+v, want the Armenian name", hy)
	}

	// Russian has no translation here, so it falls back to English rather
	// than rendering an empty line in the basket.
	ru, err := s.GetCart(ctx, userID, domain.LocaleRU)
	if err != nil {
		t.Fatal(err)
	}
	if ru[0].ProductName != "Wild Honey" {
		t.Errorf("Russian cart = %q, want the English fallback", ru[0].ProductName)
	}
}
