package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

func tableCount(t *testing.T, table string, userID int64) int {
	t.Helper()
	var n int
	if err := testPool.QueryRow(context.Background(),
		"SELECT count(*) FROM "+table+" WHERE user_id = $1", userID).Scan(&n); err != nil {
		t.Fatalf("counting %s: %v", table, err)
	}
	return n
}

// F2 (decision #97): the privacy page's sentence as a transaction. The
// test builds one customer with a full personal graph — order, address,
// session, wishlist heart, review, newsletter row — deletes the account,
// and then asserts each branch of "orders we must keep; everything else
// goes" against the database directly.
func TestDeleteAccount(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	userID := seedUserWithCart(t, "leaver@test.local", variantID, 2)

	// The checkout writes the order AND saves the address book entry.
	order, err := s.CreateOrder(ctx, userID, domain.View{Currency: domain.CurrencyUSD}, testCheckout())
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	// Refill the cart so something is there to cascade.
	if _, err := testPool.Exec(ctx,
		`INSERT INTO cart_items (user_id, variant_id, qty) VALUES ($1, $2, 1)`,
		userID, variantID); err != nil {
		t.Fatal(err)
	}

	var productID int64
	if err := testPool.QueryRow(ctx,
		`SELECT product_id FROM product_variants WHERE id = $1`, variantID).Scan(&productID); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ('live-hash', $1, now() + interval '1 day')`, []any{userID}},
		{`INSERT INTO wishlist_items (user_id, product_id) VALUES ($1, $2)`, []any{userID, productID}},
		{`INSERT INTO reviews (product_id, user_id, rating, title) VALUES ($2, $1, 5, 'Wonderful')`, []any{userID, productID}},
		{`INSERT INTO newsletter_subscribers (email, token_sha256, confirmed_at) VALUES ('leaver@test.local', 'tok', now())`, nil},
	} {
		if _, err := testPool.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatalf("seeding personal graph: %v", err)
		}
	}

	// The export reads see the graph while it exists.
	if n, err := s.CountSessions(ctx, userID); err != nil || n != 1 {
		t.Errorf("CountSessions = %d, %v; want 1", n, err)
	}
	if reviews, slugs, err := s.ReviewsByUser(ctx, userID); err != nil ||
		len(reviews) != 1 || slugs[0] != "wild-honey" {
		t.Errorf("ReviewsByUser = %v, %v, %v; want the one review on wild-honey", reviews, slugs, err)
	}

	if err := s.DeleteAccount(ctx, userID); err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}

	// The account and its personal graph are gone…
	var userCount int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE id = $1`, userID).Scan(&userCount); err != nil {
		t.Fatal(err)
	}
	if userCount != 0 {
		t.Error("user row survived")
	}
	for _, table := range []string{"sessions", "cart_items", "addresses", "wishlist_items", "reviews"} {
		if n := tableCount(t, table, userID); n != 0 {
			t.Errorf("%s: %d rows survived deletion", table, n)
		}
	}
	var newsCount int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM newsletter_subscribers WHERE email = 'leaver@test.local'`).Scan(&newsCount); err != nil {
		t.Fatal(err)
	}
	if newsCount != 0 {
		t.Error("newsletter row survived deletion")
	}

	// …while the order stays in the books, detached, with every snapshot.
	kept, err := s.GetOrder(ctx, order.ID)
	if err != nil {
		t.Fatalf("the order did not survive deletion: %v", err)
	}
	if kept.UserID != 0 {
		t.Errorf("order user id = %d, want detached (0)", kept.UserID)
	}
	if kept.TotalMinor != order.TotalMinor || kept.ShipTo == nil {
		t.Errorf("the detached order lost its snapshots: %+v", kept)
	}
	// And the admin's table still shows it — with a blank email, not a hole.
	all, err := s.ListAllOrders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].UserEmail != "" {
		t.Errorf("admin list after deletion = %+v, want the detached order with empty email", all)
	}

	// Deleting again: the account no longer exists.
	if err := s.DeleteAccount(ctx, userID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second delete: err = %v, want ErrNotFound", err)
	}
}

// A deleted admin is a demoted admin with extra steps — the #96 invariant
// holds here too.
func TestDeleteAccount_LastAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	bossID := seedUser(t, "boss@test.local")
	setRole(t, bossID, domain.RoleAdmin)

	if err := s.DeleteAccount(ctx, bossID); !errors.Is(err, domain.ErrLastAdmin) {
		t.Errorf("err = %v, want ErrLastAdmin", err)
	}

	helperID := seedUser(t, "helper@test.local")
	setRole(t, helperID, domain.RoleAdmin)
	if err := s.DeleteAccount(ctx, bossID); err != nil {
		t.Errorf("with a second admin present: %v", err)
	}
}
