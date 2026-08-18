package store_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// seedPromo writes a code straight into the tables (there is no admin write
// path in E7 — codes are seeded, by design) and returns its id.
func seedPromo(t *testing.T, code, kind string, percent int, maxRedemptions *int,
	values map[domain.Currency]domain.PromoValue) int64 {
	t.Helper()
	ctx := context.Background()

	var pct *int
	if kind == domain.PromoPercent {
		pct = &percent
	}
	var codeID int64
	err := testPool.QueryRow(ctx, `
		INSERT INTO promo_codes (code, kind, percent, max_redemptions)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		code, kind, pct, maxRedemptions).Scan(&codeID)
	if err != nil {
		t.Fatalf("seeding promo %q: %v", code, err)
	}
	for currency, v := range values {
		if _, err := testPool.Exec(ctx, `
			INSERT INTO promo_code_values (code_id, currency, amount_minor, min_subtotal_minor)
			VALUES ($1, $2, $3, $4)`,
			codeID, currency, v.AmountMinor, v.MinSubtotalMinor); err != nil {
			t.Fatalf("seeding promo value: %v", err)
		}
	}
	return codeID
}

func attachPromo(t *testing.T, userID, codeID int64) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(), `
		INSERT INTO cart_promos (user_id, code_id) VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET code_id = EXCLUDED.code_id`,
		userID, codeID); err != nil {
		t.Fatalf("attaching promo: %v", err)
	}
}

// The full promo checkout: a member's second order with a percent code —
// both discounts land, split into the two snapshot columns, the redemption
// row exists, and the cart's promo is cleared with the cart.
func TestCreateOrder_RedeemsThePromo(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 3000})
	userID := seedUserWithCart(t, "member@test.local", variantID, 1)

	// Order one makes them a member (and spends the first-delivery perk).
	if _, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout()); err != nil {
		t.Fatalf("first order: %v", err)
	}

	// Basket two, with HONEY10 applied.
	codeID := seedPromo(t, "HONEY10", domain.PromoPercent, 10, nil, nil)
	if _, err := testPool.Exec(ctx,
		`INSERT INTO cart_items (user_id, variant_id, qty) VALUES ($1, $2, 1)`,
		userID, variantID); err != nil {
		t.Fatal(err)
	}
	attachPromo(t, userID, codeID)

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("second order: %v", err)
	}

	// $30.00: member 8% = 240, code 10% = 300 — side by side, and the split
	// stored, not just the sum. Shipping $4 (under the threshold, order #2
	// so no free first delivery). Total 3000 + 400 − 540.
	if order.Totals.MemberDiscountMinor != 240 || order.Totals.PromoDiscountMinor != 300 {
		t.Errorf("discount split = %d + %d, want 240 + 300",
			order.Totals.MemberDiscountMinor, order.Totals.PromoDiscountMinor)
	}
	if order.TotalMinor != 2860 {
		t.Errorf("total = %d, want 2860", order.TotalMinor)
	}
	if order.PromoCode != "HONEY10" {
		t.Errorf("promo snapshot = %q", order.PromoCode)
	}

	// Read back through the store — the snapshot columns survive the round
	// trip, and the books still balance under the CHECK constraints.
	got, err := s.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PromoCode != "HONEY10" || got.Totals.PromoDiscountMinor != 300 {
		t.Errorf("read back: code %q, promo %d", got.PromoCode, got.Totals.PromoDiscountMinor)
	}

	var redemptions int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM promo_redemptions WHERE order_id = $1`,
		order.ID).Scan(&redemptions); err != nil {
		t.Fatal(err)
	}
	if redemptions != 1 {
		t.Errorf("%d redemption rows, want 1", redemptions)
	}
	var cartPromos int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM cart_promos WHERE user_id = $1`,
		userID).Scan(&cartPromos); err != nil {
		t.Fatal(err)
	}
	if cartPromos != 0 {
		t.Error("cart promo survived the checkout")
	}
}

// A code that dies between apply and checkout is a refusal, not a silent
// repricing: the customer approved a number this order would not match.
func TestCreateOrder_RefusesAStaleCode(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 3000})
	userID := seedUserWithCart(t, "stale@test.local", variantID, 1)
	codeID := seedPromo(t, "AUGUST", domain.PromoPercent, 15, nil, nil)
	attachPromo(t, userID, codeID)

	// The family deactivates the code while the basket sits.
	if _, err := testPool.Exec(ctx,
		`UPDATE promo_codes SET active = FALSE WHERE id = $1`, codeID); err != nil {
		t.Fatal(err)
	}

	_, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if !errors.Is(err, domain.ErrPromoInvalid) {
		t.Fatalf("err = %v, want ErrPromoInvalid", err)
	}

	// The refusal left the world untouched: no order, stock intact, and the
	// cart still full — the transaction rolled back whole.
	var orders int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM orders WHERE user_id = $1`, userID).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if orders != 0 {
		t.Errorf("%d orders created despite the refusal", orders)
	}
}

// Cancelling an order releases its redemption — the promo equivalent of
// restoring stock — while the order keeps its promo_code snapshot.
func TestCancelReleasesTheRedemption(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 3000})
	userID := seedUserWithCart(t, "undo@test.local", variantID, 1)
	one := 1
	codeID := seedPromo(t, "ONEUSE", domain.PromoPercent, 10, &one, nil)
	attachPromo(t, userID, codeID)

	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatal(err)
	}

	// Spent: the shopper's own view says used, the global count says full.
	promo, err := s.PromoForUser(ctx, "oneuse", userID) // lowercase finds it too
	if err != nil {
		t.Fatal(err)
	}
	if !promo.UsedByShopper || promo.Redemptions != 1 {
		t.Fatalf("before cancel: used=%v redemptions=%d", promo.UsedByShopper, promo.Redemptions)
	}

	if _, err := s.UpdateOrderStatus(ctx, order.ID, domain.OrderCancelled); err != nil {
		t.Fatal(err)
	}

	promo, err = s.PromoForUser(ctx, "ONEUSE", userID)
	if err != nil {
		t.Fatal(err)
	}
	if promo.UsedByShopper || promo.Redemptions != 0 {
		t.Errorf("after cancel: used=%v redemptions=%d, want free again",
			promo.UsedByShopper, promo.Redemptions)
	}
	// The receipt still names the code — history is not rewritten, only the
	// redemption released.
	got, err := s.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.PromoCode != "ONEUSE" {
		t.Errorf("cancelled order lost its snapshot: %q", got.PromoCode)
	}
}

// THE concurrency test the plan asked for, in the oversell test's image:
// ten users, ten parallel checkouts, one code with max_redemptions = 1 —
// exactly one order gets the discount, and the losers are REFUSED (the
// customer sees why) rather than silently charged full price.
func TestCreateOrder_ParallelCheckoutsCannotOverRedeem(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	const shoppers = 10
	variantID := seedPricedProduct(t, "honey", "HON-1", 100, domain.Money{domain.CurrencyUSD: 3000})
	one := 1
	codeID := seedPromo(t, "GOLDRUSH", domain.PromoPercent, 10, &one, nil)

	userIDs := make([]int64, shoppers)
	for i := range userIDs {
		userIDs[i] = seedUserWithCart(t,
			fmt.Sprintf("racer%d@test.local", i), variantID, 1)
		attachPromo(t, userIDs[i], codeID)
	}

	var wg sync.WaitGroup
	errs := make([]error, shoppers)
	for i := range userIDs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = s.CreateOrder(ctx, userIDs[i], domain.View{Currency: domain.CurrencyUSD}, testCheckout())
		}(i)
	}
	wg.Wait()

	var won, refused int
	for _, err := range errs {
		switch {
		case err == nil:
			won++
		case errors.Is(err, domain.ErrPromoInvalid):
			refused++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if won != 1 || refused != shoppers-1 {
		t.Errorf("won=%d refused=%d, want exactly 1 and %d", won, refused, shoppers-1)
	}

	// The database agrees with the arithmetic: one redemption, one order
	// carrying the discount.
	var redemptions, discounted int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM promo_redemptions WHERE code_id = $1`, codeID).Scan(&redemptions); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM orders WHERE promo_discount_minor > 0`).Scan(&discounted); err != nil {
		t.Fatal(err)
	}
	if redemptions != 1 || discounted != 1 {
		t.Errorf("redemptions=%d discounted orders=%d, want 1 and 1", redemptions, discounted)
	}
}

// The perk pair in one place: order one ships free with no member discount,
// order two pays shipping and gets 8% — "first order ships free" and "8%
// less on every order after the first" read straight off the sign-in screen.
func TestHiveClubPerksAcrossTwoOrders(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 3000})
	userID := seedUserWithCart(t, "newbee@test.local", variantID, 1)

	first, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatal(err)
	}
	if first.Totals.ShippingMinor != 0 || first.Totals.MemberDiscountMinor != 0 {
		t.Errorf("first order: shipping=%d member=%d, want 0 and 0",
			first.Totals.ShippingMinor, first.Totals.MemberDiscountMinor)
	}

	if _, err := testPool.Exec(ctx,
		`INSERT INTO cart_items (user_id, variant_id, qty) VALUES ($1, $2, 1)`,
		userID, variantID); err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatal(err)
	}
	if second.Totals.ShippingMinor != 400 || second.Totals.MemberDiscountMinor != 240 {
		t.Errorf("second order: shipping=%d member=%d, want 400 and 240",
			second.Totals.ShippingMinor, second.Totals.MemberDiscountMinor)
	}
}
