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

// testCheckout is a complete, valid checkout input for tests whose subject
// is NOT the checkout itself (stock, counters, currency). Tests about the
// checkout build their own.
func testCheckout() domain.CheckoutInput {
	return domain.CheckoutInput{
		Address: domain.Address{
			FirstName: "Anahit", LastName: "Sargsyan", Phone: "+374 91 000000",
			Street: "14 Abovyan St, apt 6", City: "Yerevan",
			PostalCode: "0009", Country: "AM",
		},
		PaymentMethod: domain.PayCard,
	}
}

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
		), variant AS (
			INSERT INTO product_variants (product_id, sku, label, stock_qty)
			SELECT id, 'HON-1', '700 g', $1 FROM prod
			RETURNING id
		)
		INSERT INTO variant_prices (variant_id, currency, price_minor)
		SELECT id, 'USD', 950000 FROM variant
		RETURNING variant_id`, stock).Scan(&variantID)
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

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
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

	_, err := s.CreateOrder(context.Background(), userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
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

	_, err := s.CreateOrder(context.Background(), userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Errorf("err = %v, want ErrInsufficientStock", err)
	}
	// The typed error must carry the count read under the lock — it is what
	// the API forwards so the customer learns how many they CAN buy.
	var short *domain.StockShortError
	if !errors.As(err, &short) {
		t.Fatalf("err = %T, want *domain.StockShortError", err)
	}
	if short.Available != 2 {
		t.Errorf("Available = %d, want 2", short.Available)
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
			_, err := s.CreateOrder(ctx, uid, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
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

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
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

// F2: the customer's self-service cancel. Same side effects as the
// admin's cancel (they share applyOrderStatusTx), so what THIS test pins
// is the two gates in front of them: ownership answers ErrNotFound
// (existence-hiding), and the pending-only window answers
// ErrTooLateToCancel with nothing changed.
func TestCancelOrderByCustomer(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	userID := seedUserWithCart(t, "regretter@test.local", variantID, 4)
	strangerID := seedUser(t, "stranger@test.local")

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// A stranger's attempt does not even learn the order exists.
	if _, err := s.CancelOrderByCustomer(ctx, strangerID, order.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("stranger: err = %v, want ErrNotFound", err)
	}

	// The owner, while pending: cancelled, stock back, history recorded.
	got, err := s.CancelOrderByCustomer(ctx, userID, order.ID)
	if err != nil {
		t.Fatalf("cancelling: %v", err)
	}
	if got.Status != domain.OrderCancelled {
		t.Errorf("status = %q, want cancelled", got.Status)
	}
	var stock int
	if err := testPool.QueryRow(ctx,
		`SELECT stock_qty FROM product_variants WHERE id = $1`, variantID).Scan(&stock); err != nil {
		t.Fatal(err)
	}
	if stock != 10 {
		t.Errorf("stock after cancel = %d, want 10", stock)
	}
	read, err := s.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(read.Events); n != 2 || read.Events[n-1].Status != domain.OrderCancelled {
		t.Errorf("events = %+v, want pending then cancelled", read.Events)
	}

	// A CONFIRMED order is past the customer's window — refused with
	// nothing changed, even though the machine itself allows the admin
	// that same arrow.
	userID2 := seedUserWithCart(t, "slowpoke@test.local", variantID, 2)
	order2, err := s.CreateOrder(ctx, userID2, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if _, err := s.UpdateOrderStatus(ctx, order2.ID, domain.OrderConfirmed); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	if _, err := s.CancelOrderByCustomer(ctx, userID2, order2.ID); !errors.Is(err, domain.ErrTooLateToCancel) {
		t.Errorf("confirmed: err = %v, want ErrTooLateToCancel", err)
	}
	read2, err := s.GetOrder(ctx, order2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read2.Status != domain.OrderConfirmed {
		t.Errorf("status = %q — the refused cancel must change nothing", read2.Status)
	}
}

// F2: the payment machine's write path. The interesting assertions are the
// re-reads — GetOrder after each flip proves the change was COMMITTED, not
// just present on the struct the method handed back.
func TestUpdateOrderPaymentStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	userID := seedUserWithCart(t, "payer@test.local", variantID, 2)

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if order.PaymentStatus != domain.PaymentUnpaid {
		t.Fatalf("new order payment = %q, want unpaid", order.PaymentStatus)
	}

	// Refunding money never taken is refused before any write.
	_, err = s.UpdateOrderPaymentStatus(ctx, order.ID, domain.PaymentRefunded)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("unpaid → refunded: err = %v, want ErrInvalidTransition", err)
	}

	if _, err := s.UpdateOrderPaymentStatus(ctx, order.ID, domain.PaymentPaid); err != nil {
		t.Fatalf("marking paid: %v", err)
	}
	got, err := s.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PaymentStatus != domain.PaymentPaid {
		t.Errorf("after mark-paid, stored payment = %q, want paid", got.PaymentStatus)
	}

	// Orthogonality: flipping payment must not have touched order status.
	if got.Status != domain.OrderPending {
		t.Errorf("order status changed to %q by a payment flip", got.Status)
	}

	if _, err := s.UpdateOrderPaymentStatus(ctx, order.ID, domain.PaymentRefunded); err != nil {
		t.Fatalf("refunding: %v", err)
	}
	got, err = s.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PaymentStatus != domain.PaymentRefunded {
		t.Errorf("after refund, stored payment = %q, want refunded", got.PaymentStatus)
	}

	// Refunded is terminal — no erasing, no re-paying.
	_, err = s.UpdateOrderPaymentStatus(ctx, order.ID, domain.PaymentPaid)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("refunded → paid: err = %v, want ErrInvalidTransition", err)
	}

	// And a missing order is a missing order.
	_, err = s.UpdateOrderPaymentStatus(ctx, 99999, domain.PaymentPaid)
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("unknown id: err = %v, want ErrNotFound", err)
	}
}

// products.sales_count is denormalized data (migration 000010): it is only
// correct because the checkout transaction maintains it, so the increment
// needs a test the way an aggregate query would not.
func TestCreateOrder_IncrementsSalesCount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 20)
	ctx0 := context.Background()

	// A SECOND variant of the same product: one cart holding both must move
	// the counter once, by the summed quantity, not twice.
	var variant2ID int64
	if err := testPool.QueryRow(ctx0, `
		WITH variant AS (
			INSERT INTO product_variants (product_id, sku, label, stock_qty)
			SELECT product_id, 'HON-2', '350 g', 20
			FROM product_variants WHERE id = $1
			RETURNING id
		)
		INSERT INTO variant_prices (variant_id, currency, price_minor)
		SELECT id, 'USD', 520000 FROM variant
		RETURNING variant_id`, variantID).Scan(&variant2ID); err != nil {
		t.Fatalf("seeding second variant: %v", err)
	}

	userID := seedUserWithCart(t, "counter@test.local", variantID, 3)
	if _, err := testPool.Exec(ctx,
		`INSERT INTO cart_items (user_id, variant_id, qty) VALUES ($1, $2, 2)`,
		userID, variant2ID); err != nil {
		t.Fatalf("adding second cart line: %v", err)
	}

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	salesCount := func() int {
		t.Helper()
		var n int
		if err := testPool.QueryRow(ctx,
			`SELECT sales_count FROM products WHERE slug = 'wild-honey'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	if got := salesCount(); got != 5 {
		t.Errorf("sales_count = %d, want 5 (3 + 2 across two variants of one product)", got)
	}

	// Cancelling deliberately does NOT decrement — "most loved" measures
	// interest over time, and this pins that as a decision rather than
	// leaving a missing UPDATE to look like an oversight.
	if _, err := s.UpdateOrderStatus(ctx, order.ID, domain.OrderCancelled); err != nil {
		t.Fatalf("cancelling: %v", err)
	}
	if got := salesCount(); got != 5 {
		t.Errorf("sales_count after cancel = %d, want 5 (cancellation does not undo interest)", got)
	}
}

// A2 (decision log #85): every transition leaves a recorded fact. The
// timeline must grow with the order — pending at creation, each step of the
// state machine after — and come back on every read, oldest first.
func TestOrderStatusEvents_RecordedAndAttached(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	userID := seedUserWithCart(t, "history@test.local", variantID, 1)

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if len(order.Events) != 1 || order.Events[0].Status != domain.OrderPending {
		t.Fatalf("events after create = %+v, want one pending", order.Events)
	}
	// The first event carries the ORDER's timestamp — "placed" and
	// "created" are the same instant by construction, not by luck.
	if !order.Events[0].CreatedAt.Equal(order.CreatedAt) {
		t.Errorf("pending event at %v, order created at %v", order.Events[0].CreatedAt, order.CreatedAt)
	}

	if _, err := s.UpdateOrderStatus(ctx, order.ID, domain.OrderConfirmed); err != nil {
		t.Fatalf("confirming: %v", err)
	}
	if _, err := s.UpdateOrderStatus(ctx, order.ID, domain.OrderShipped); err != nil {
		t.Fatalf("shipping: %v", err)
	}

	got, err := s.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	want := []string{domain.OrderPending, domain.OrderConfirmed, domain.OrderShipped}
	if len(got.Events) != len(want) {
		t.Fatalf("events = %+v, want %v", got.Events, want)
	}
	for i, w := range want {
		if got.Events[i].Status != w {
			t.Errorf("events[%d] = %q, want %q", i, got.Events[i].Status, w)
		}
	}

	// The list read attaches the same history.
	orders, err := s.ListOrdersByUser(ctx, userID)
	if err != nil {
		t.Fatalf("ListOrdersByUser: %v", err)
	}
	if len(orders) != 1 || len(orders[0].Events) != 3 {
		t.Errorf("listed order events = %+v, want 3", orders)
	}
}

// A2 (decision log #86): reorder merges a past order into the cart and
// reports each line's fate — full add, capped by stock, or skipped.
func TestReorder(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	userID := seedUserWithCart(t, "again@test.local", variantID, 3)

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	// Checkout cleared the cart and left stock at 7.

	cartQty := func() int {
		t.Helper()
		var n int
		if err := testPool.QueryRow(ctx, `
			SELECT COALESCE(sum(qty), 0) FROM cart_items WHERE user_id = $1`,
			userID).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("full add into an empty cart", func(t *testing.T) {
		res, err := s.Reorder(ctx, userID, order.ID)
		if err != nil {
			t.Fatalf("Reorder: %v", err)
		}
		if len(res.Lines) != 1 || res.Lines[0].Qty != 3 || res.Lines[0].Issue != "" {
			t.Fatalf("lines = %+v, want one full add of 3", res.Lines)
		}
		if got := cartQty(); got != 3 {
			t.Errorf("cart qty = %d, want 3", got)
		}
	})

	t.Run("quantities merge, not replace", func(t *testing.T) {
		if _, err := s.Reorder(ctx, userID, order.ID); err != nil {
			t.Fatalf("second Reorder: %v", err)
		}
		if got := cartQty(); got != 6 {
			t.Errorf("cart qty = %d, want 6 (3 + 3)", got)
		}
	})

	t.Run("capped by stock and reported as reduced", func(t *testing.T) {
		// Stock 7, cart already 6 → room for 1 of the requested 3.
		res, err := s.Reorder(ctx, userID, order.ID)
		if err != nil {
			t.Fatalf("third Reorder: %v", err)
		}
		if res.Lines[0].Qty != 1 || res.Lines[0].Issue != domain.ReorderReduced {
			t.Errorf("line = %+v, want qty 1 reduced", res.Lines[0])
		}
		if got := cartQty(); got != 7 {
			t.Errorf("cart qty = %d, want 7 (capped at stock)", got)
		}
	})

	t.Run("no room left is out_of_stock and adds nothing", func(t *testing.T) {
		res, err := s.Reorder(ctx, userID, order.ID)
		if err != nil {
			t.Fatalf("fourth Reorder: %v", err)
		}
		if res.Lines[0].Qty != 0 || res.Lines[0].Issue != domain.ReorderOutOfStock {
			t.Errorf("line = %+v, want out_of_stock", res.Lines[0])
		}
		if got := cartQty(); got != 7 {
			t.Errorf("cart qty = %d, want 7 still", got)
		}
	})

	t.Run("a retired product is unavailable", func(t *testing.T) {
		if _, err := testPool.Exec(ctx,
			`UPDATE products SET is_active = false WHERE slug = 'wild-honey'`); err != nil {
			t.Fatal(err)
		}
		res, err := s.Reorder(ctx, userID, order.ID)
		if err != nil {
			t.Fatalf("Reorder on retired product: %v", err)
		}
		if res.Lines[0].Issue != domain.ReorderUnavailable {
			t.Errorf("line = %+v, want unavailable", res.Lines[0])
		}
	})

	t.Run("a stranger's order id is ErrNotFound", func(t *testing.T) {
		var strangerID int64
		if err := testPool.QueryRow(ctx, `
			INSERT INTO users (email, password_hash) VALUES ('other@test.local', 'x')
			RETURNING id`).Scan(&strangerID); err != nil {
			t.Fatal(err)
		}
		if _, err := s.Reorder(ctx, strangerID, order.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}
