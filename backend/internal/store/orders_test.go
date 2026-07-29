package store_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// seedCatalog inserts one product with one variant and returns the variant id.
func seedCatalog(t *testing.T, stock int) int64 {
	t.Helper()
	ctx := context.Background()

	var variantID int64
	err := testPool.QueryRow(ctx, `
		WITH cat AS (
			INSERT INTO categories (slug, name) VALUES ('honey', 'Honey') RETURNING id
		), prod AS (
			INSERT INTO products (category_id, slug, name)
			SELECT id, 'wild-honey', 'Wild Honey' FROM cat RETURNING id
		)
		INSERT INTO product_variants (product_id, sku, label, price_minor, stock_qty)
		SELECT id, 'HON-1', '700 g', 950000, $1 FROM prod
		RETURNING id`, stock).Scan(&variantID)
	if err != nil {
		t.Fatalf("seeding catalog: %v", err)
	}
	return variantID
}

func seedUserWithCart(t *testing.T, email string, variantID int64, qty int) int64 {
	t.Helper()
	ctx := context.Background()

	var userID int64
	err := testPool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, 'x') RETURNING id`,
		email).Scan(&userID)
	if err != nil {
		t.Fatalf("seeding user: %v", err)
	}
	if _, err := testPool.Exec(ctx, `
		INSERT INTO cart_items (user_id, variant_id, qty) VALUES ($1, $2, $3)`,
		userID, variantID, qty); err != nil {
		t.Fatalf("seeding cart: %v", err)
	}
	return userID
}

func TestCreateOrder_HappyPath(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	userID := seedUserWithCart(t, "buyer@test.local", variantID, 3)

	order, err := s.CreateOrder(ctx, userID)
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	if order.Status != domain.OrderPending {
		t.Errorf("status = %q, want pending", order.Status)
	}
	if order.TotalMinor != 3*950000 {
		t.Errorf("total = %d, want %d", order.TotalMinor, 3*950000)
	}
	if len(order.Items) != 1 || order.Items[0].Qty != 3 {
		t.Errorf("unexpected items: %+v", order.Items)
	}

	// Stock decremented and cart cleared — check the DB, not the return value.
	var stock, cartCount int
	if err := testPool.QueryRow(ctx,
		`SELECT stock_qty FROM product_variants WHERE id = $1`, variantID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 7 {
		t.Errorf("stock = %d, want 7", stock)
	}
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM cart_items WHERE user_id = $1`, userID).Scan(&cartCount); err != nil {
		t.Fatal(err)
	}
	if cartCount != 0 {
		t.Errorf("cart not cleared: %d items left", cartCount)
	}
}

func TestCreateOrder_EmptyCart(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)

	var userID int64
	if err := testPool.QueryRow(context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ('empty@test.local', 'x') RETURNING id`,
	).Scan(&userID); err != nil {
		t.Fatal(err)
	}

	_, err := s.CreateOrder(context.Background(), userID)
	if !errors.Is(err, domain.ErrEmptyCart) {
		t.Errorf("err = %v, want ErrEmptyCart", err)
	}
}

func TestCreateOrder_InsufficientStock(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)

	variantID := seedCatalog(t, 2)
	userID := seedUserWithCart(t, "greedy@test.local", variantID, 5)

	_, err := s.CreateOrder(context.Background(), userID)
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Errorf("err = %v, want ErrInsufficientStock", err)
	}

	// Nothing must have changed — the transaction rolled back.
	var stock int
	if err := testPool.QueryRow(context.Background(),
		`SELECT stock_qty FROM product_variants WHERE id = $1`, variantID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 2 {
		t.Errorf("stock = %d, want 2 (rollback failed?)", stock)
	}
}

// The manual honey race from Phase 5, now permanent: 10 buyers, stock 3 —
// exactly 3 orders may succeed and stock must end at exactly 0.
func TestCreateOrder_ConcurrentCheckoutsDoNotOversell(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	const stock = 3
	const buyers = 10

	variantID := seedCatalog(t, stock)
	userIDs := make([]int64, buyers)
	for i := range userIDs {
		userIDs[i] = seedUserWithCart(t, string(rune('a'+i))+"@race.local", variantID, 1)
	}

	var succeeded, refused atomic.Int64
	var wg sync.WaitGroup
	for _, uid := range userIDs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.CreateOrder(ctx, uid)
			switch {
			case err == nil:
				succeeded.Add(1)
			case errors.Is(err, domain.ErrInsufficientStock):
				refused.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if got := succeeded.Load(); got != stock {
		t.Errorf("%d orders succeeded, want exactly %d", got, stock)
	}
	if got := refused.Load(); got != buyers-stock {
		t.Errorf("%d refused, want %d", got, buyers-stock)
	}

	var finalStock int
	if err := testPool.QueryRow(ctx,
		`SELECT stock_qty FROM product_variants WHERE id = $1`, variantID).Scan(&finalStock); err != nil {
		t.Fatal(err)
	}
	if finalStock != 0 {
		t.Errorf("final stock = %d, want 0", finalStock)
	}

	var orderCount int
	if err := testPool.QueryRow(ctx, `SELECT count(*) FROM orders`).Scan(&orderCount); err != nil {
		t.Fatal(err)
	}
	if orderCount != stock {
		t.Errorf("%d orders in db, want %d", orderCount, stock)
	}
}

func TestUpdateOrderStatus_CancelRestoresStock(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	userID := seedUserWithCart(t, "canceller@test.local", variantID, 4)

	order, err := s.CreateOrder(ctx, userID)
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	if _, err := s.UpdateOrderStatus(ctx, order.ID, domain.OrderCancelled); err != nil {
		t.Fatalf("cancelling: %v", err)
	}

	var stock int
	if err := testPool.QueryRow(ctx,
		`SELECT stock_qty FROM product_variants WHERE id = $1`, variantID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 10 {
		t.Errorf("stock after cancel = %d, want 10", stock)
	}

	// And the state machine still refuses nonsense.
	_, err = s.UpdateOrderStatus(ctx, order.ID, domain.OrderShipped)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("err = %v, want ErrInvalidTransition", err)
	}
}
