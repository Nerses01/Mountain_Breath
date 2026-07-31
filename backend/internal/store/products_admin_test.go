package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

func TestCreateProduct_Transactional(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	var categoryID int64
	if err := testPool.QueryRow(ctx,
		`INSERT INTO categories (slug, name) VALUES ('jam', 'Jam') RETURNING id`,
	).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}

	p := domain.Product{
		CategoryID: categoryID, Slug: "berry-jam", Name: "Berry Jam", IsActive: true,
		Variants: []domain.ProductVariant{
			{SKU: "JAM-300", Label: "300 g", PriceMinor: 250000, StockQty: 5},
			{SKU: "JAM-600", Label: "600 g", PriceMinor: 450000, StockQty: 2},
		},
	}
	if err := s.CreateProduct(ctx, &p); err != nil {
		t.Fatalf("CreateProduct: %v", err)
	}
	if p.ID == 0 || p.Variants[0].ID == 0 {
		t.Errorf("ids not filled in: product=%d variant=%d", p.ID, p.Variants[0].ID)
	}

	// Second product reusing an SKU must fail AND leave no orphan product
	// behind — the transaction is the point of this test.
	dup := domain.Product{
		CategoryID: categoryID, Slug: "other-jam", Name: "Other", IsActive: true,
		Variants: []domain.ProductVariant{{SKU: "JAM-300", Label: "1 kg", PriceMinor: 100, StockQty: 1}},
	}
	err := s.CreateProduct(ctx, &dup)
	if !errors.Is(err, domain.ErrSKUTaken) {
		t.Fatalf("err = %v, want ErrSKUTaken", err)
	}

	var productCount int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM products`).Scan(&productCount); err != nil {
		t.Fatal(err)
	}
	if productCount != 1 {
		t.Errorf("product count = %d, want 1 (rollback failed — orphan product left behind)", productCount)
	}

	// Unknown category maps to the right sentinel.
	badCat := domain.Product{
		CategoryID: 99999, Slug: "ghost", Name: "Ghost", IsActive: true,
		Variants: []domain.ProductVariant{{SKU: "GH-1", Label: "1", PriceMinor: 1, StockQty: 0}},
	}
	if err := s.CreateProduct(ctx, &badCat); !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Errorf("err = %v, want ErrCategoryNotFound", err)
	}
}

func TestAdminListIncludesInactive(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 5) // one active product
	_ = variantID
	if _, err := testPool.Exec(ctx, `
		INSERT INTO products (category_id, slug, name, is_active)
		SELECT category_id, 'retired', 'Retired Product', FALSE FROM products LIMIT 1`); err != nil {
		t.Fatal(err)
	}

	public, _, err := s.ListProducts(ctx, domain.ProductFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatal(err)
	}
	admin, _, err := s.ListProducts(ctx, domain.ProductFilter{Page: 1, PerPage: 10, IncludeInactive: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(public) != 1 {
		t.Errorf("public list has %d products, want 1", len(public))
	}
	if len(admin) != 2 {
		t.Errorf("admin list has %d products, want 2", len(admin))
	}
}
