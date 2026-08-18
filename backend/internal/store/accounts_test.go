package store_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// (seedUser lives in reviews_test.go — same package, one helper.)

// ── Password reset ────────────────────────────────────────────────────────

// The plan's test, by name: a reset token works once, and not after expiry.
func TestPasswordReset_SingleUseAndExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	userID := seedUser(t, "reset@test.local")

	t.Run("a token is spent by use", func(t *testing.T) {
		if err := s.CreatePasswordReset(ctx, userID, "token-one", time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := s.ConsumePasswordReset(ctx, "token-one", "new-hash-1"); err != nil {
			t.Fatalf("first use: %v", err)
		}
		// The second use is indistinguishable from a token that never was.
		if err := s.ConsumePasswordReset(ctx, "token-one", "new-hash-2"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("second use: %v, want ErrNotFound", err)
		}
		var hash string
		if err := testPool.QueryRow(ctx,
			`SELECT password_hash FROM users WHERE id = $1`, userID).Scan(&hash); err != nil {
			t.Fatal(err)
		}
		if hash != "new-hash-1" {
			t.Errorf("hash = %q — the spent token changed the password again", hash)
		}
	})

	t.Run("an expired token never works", func(t *testing.T) {
		if err := s.CreatePasswordReset(ctx, userID, "token-two", time.Now().Add(-time.Minute)); err != nil {
			t.Fatal(err)
		}
		if err := s.ConsumePasswordReset(ctx, "token-two", "new-hash-3"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("expired token: %v, want ErrNotFound", err)
		}
	})

	t.Run("a new request retires the old link", func(t *testing.T) {
		if err := s.CreatePasswordReset(ctx, userID, "token-three", time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := s.CreatePasswordReset(ctx, userID, "token-four", time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		if err := s.ConsumePasswordReset(ctx, "token-three", "x"); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("superseded token still worked: %v", err)
		}
		if err := s.ConsumePasswordReset(ctx, "token-four", "new-hash-4"); err != nil {
			t.Errorf("the newest link failed: %v", err)
		}
	})
}

// A reset is a "someone may have my password" event, so it revokes every
// session — the stolen cookie dies with the stolen password.
func TestPasswordReset_RevokesSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	userID := seedUser(t, "revoke@test.local")
	if err := s.CreateSession(ctx, "session-token", userID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetUserBySession(ctx, "session-token"); err != nil {
		t.Fatalf("session should work before the reset: %v", err)
	}

	if err := s.CreatePasswordReset(ctx, userID, "reset-token", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := s.ConsumePasswordReset(ctx, "reset-token", "new-hash"); err != nil {
		t.Fatal(err)
	}

	if _, err := s.GetUserBySession(ctx, "session-token"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("session survived the reset: %v", err)
	}
}

// ── Wishlist ──────────────────────────────────────────────────────────────

// The other named test: save-for-later moves exactly one row each way.
func TestSaveForLater_MovesExactlyOneRowEachWay(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 1400})
	userID := seedUserWithCart(t, "later@test.local", variantID, 2)

	if err := s.SaveForLater(ctx, userID, variantID); err != nil {
		t.Fatalf("SaveForLater: %v", err)
	}

	var cartRows, wishRows int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM cart_items WHERE user_id = $1`, userID).Scan(&cartRows); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM wishlist_items WHERE user_id = $1`, userID).Scan(&wishRows); err != nil {
		t.Fatal(err)
	}
	if cartRows != 0 || wishRows != 1 {
		t.Errorf("cart=%d wishlist=%d, want 0 and 1", cartRows, wishRows)
	}

	// The second identical move finds nothing to move — a TRANSFER, not an
	// idempotent state write — and the wishlist is untouched by the refusal.
	if err := s.SaveForLater(ctx, userID, variantID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("second move: %v, want ErrNotFound", err)
	}
}

func TestWishlist_ListsCardsNewestFirst(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	honeyVariant := seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 1400})
	pollenVariant := seedPricedProduct(t, "pollen", "POL-1", 10, domain.Money{domain.CurrencyUSD: 1600})
	_ = honeyVariant
	_ = pollenVariant
	userID := seedUser(t, "hearts@test.local")

	var honeyID, pollenID int64
	if err := testPool.QueryRow(ctx, `SELECT id FROM products WHERE slug = 'honey'`).Scan(&honeyID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `SELECT id FROM products WHERE slug = 'pollen'`).Scan(&pollenID); err != nil {
		t.Fatal(err)
	}

	// Heart both (honey twice — idempotent), then retire pollen.
	for _, id := range []int64{honeyID, honeyID, pollenID} {
		if err := s.AddWishlistItem(ctx, userID, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := testPool.Exec(ctx,
		`UPDATE products SET is_active = FALSE WHERE id = $1`, pollenID); err != nil {
		t.Fatal(err)
	}

	items, err := s.ListWishlist(ctx, userID, domain.View{})
	if err != nil {
		t.Fatal(err)
	}
	// One row: hearting twice was one fact, and the retired jar dropped out
	// of the LIST while keeping its row for a possible return.
	if len(items) != 1 || items[0].Slug != "honey" {
		t.Fatalf("wishlist = %+v, want just honey", items)
	}
	if len(items[0].Variants) == 0 || items[0].Variants[0].PriceMinor != 1400 {
		t.Error("wishlist card came back without its price — it must render like any grid card")
	}
	// A3: every card carries WHEN it was hearted.
	if items[0].SavedAt.IsZero() {
		t.Error("saved_at missing — the canvas's \"saved N ago\" line has nothing to render")
	}

	var rawRows int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM wishlist_items WHERE user_id = $1`, userID).Scan(&rawRows); err != nil {
		t.Fatal(err)
	}
	if rawRows != 2 {
		t.Errorf("raw rows = %d — the inactive product's heart should survive", rawRows)
	}
}

// ── Address book ──────────────────────────────────────────────────────────

func TestAddressBook_DefaultJuggling(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	userID := seedUser(t, "book@test.local")
	addr := domain.Address{
		FirstName: "Anahit", LastName: "Sargsyan", Phone: "+374",
		Street: "14 Abovyan St", City: "Yerevan", PostalCode: "0009", Country: "AM",
	}

	// The first entry is default whatever the flag said.
	home, err := s.CreateAddress(ctx, userID, domain.AddressEntry{Label: "Home", Address: addr})
	if err != nil {
		t.Fatal(err)
	}
	if !home.IsDefault {
		t.Error("first address did not become the default")
	}

	// A second entry claiming the default demotes the first — the partial
	// unique index would reject two defaults, so the store must swap inside
	// one transaction.
	office := addr
	office.Street = "2 Mashtots Ave"
	second, err := s.CreateAddress(ctx, userID,
		domain.AddressEntry{Label: "Office", IsDefault: true, Address: office})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := s.ListAddresses(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("%d entries, want 2", len(entries))
	}
	// Default first, per the ORDER BY.
	if entries[0].ID != second.ID || !entries[0].IsDefault || entries[1].IsDefault {
		t.Errorf("default juggling failed: %+v", entries)
	}

	// The checkout's prefill read agrees with the book.
	prefill, err := s.DefaultAddress(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if prefill.Street != "2 Mashtots Ave" {
		t.Errorf("prefill street = %q, want the office", prefill.Street)
	}

	// Deleting someone else's address is a 404-shaped no.
	if err := s.DeleteAddress(ctx, userID+1, home.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("foreign delete: %v, want ErrNotFound", err)
	}
	if err := s.DeleteAddress(ctx, userID, home.ID); err != nil {
		t.Fatal(err)
	}
}

// ── OAuth identities ──────────────────────────────────────────────────────

func TestFindOrCreateOAuthUser(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	t.Run("a stranger becomes a passwordless customer", func(t *testing.T) {
		u, err := s.FindOrCreateOAuthUser(ctx, "google", "sub-1", "new@gmail.example")
		if err != nil {
			t.Fatal(err)
		}
		if u.PasswordHash != "" || u.Role != domain.RoleCustomer {
			t.Errorf("minted user = %+v", u)
		}
		// Same subject again resolves to the SAME account — the identity is
		// the subject, not the email.
		again, err := s.FindOrCreateOAuthUser(ctx, "google", "sub-1", "changed@gmail.example")
		if err != nil {
			t.Fatal(err)
		}
		if again.ID != u.ID {
			t.Errorf("subject resolved to a different account: %d then %d", u.ID, again.ID)
		}
		if again.Email != "new@gmail.example" {
			t.Errorf("account email followed the provider's: %q", again.Email)
		}
	})

	t.Run("a verified email links to the password account that owns it", func(t *testing.T) {
		existing := seedUser(t, "member@test.local")
		u, err := s.FindOrCreateOAuthUser(ctx, "google", "sub-2", "member@test.local")
		if err != nil {
			t.Fatal(err)
		}
		if u.ID != existing {
			t.Errorf("linked to %d, want the existing account %d", u.ID, existing)
		}
		var identities int
		if err := testPool.QueryRow(ctx,
			`SELECT count(*) FROM oauth_identities WHERE user_id = $1`, existing).Scan(&identities); err != nil {
			t.Fatal(err)
		}
		if identities != 1 {
			t.Errorf("%d identities on the account, want 1", identities)
		}
	})
}

// A3: "Add all to cart" is the reorder merge's sibling — one of each saved
// product, first variant with room, partial success reported per line.
func TestAddWishlistToCart(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	seedPricedProduct(t, "honey", "HON-1", 10, domain.Money{domain.CurrencyUSD: 1400})
	seedPricedProduct(t, "comb", "COMB-1", 0, domain.Money{domain.CurrencyUSD: 2200})
	userID := seedUser(t, "addall@test.local")

	var honeyID, combID int64
	if err := testPool.QueryRow(ctx, `SELECT id FROM products WHERE slug = 'honey'`).Scan(&honeyID); err != nil {
		t.Fatal(err)
	}
	if err := testPool.QueryRow(ctx, `SELECT id FROM products WHERE slug = 'comb'`).Scan(&combID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{honeyID, combID} {
		if err := s.AddWishlistItem(ctx, userID, id); err != nil {
			t.Fatal(err)
		}
	}

	res, err := s.AddWishlistToCart(ctx, userID)
	if err != nil {
		t.Fatalf("AddWishlistToCart: %v", err)
	}
	if len(res.Lines) != 2 {
		t.Fatalf("lines = %+v, want 2", res.Lines)
	}

	byName := map[string]domain.ReorderLine{}
	for _, l := range res.Lines {
		byName[l.Name] = l
	}
	if l := byName["honey"]; l.Qty != 1 || l.Issue != "" {
		t.Errorf("honey line = %+v, want one added cleanly", l)
	}
	if l := byName["comb"]; l.Qty != 0 || l.Issue != domain.ReorderOutOfStock {
		t.Errorf("comb line = %+v, want out_of_stock skip", l)
	}

	// Running it again MERGES: the honey line becomes qty 2.
	if _, err := s.AddWishlistToCart(ctx, userID); err != nil {
		t.Fatal(err)
	}
	var qty int
	if err := testPool.QueryRow(ctx, `
		SELECT COALESCE(sum(qty), 0) FROM cart_items WHERE user_id = $1`,
		userID).Scan(&qty); err != nil {
		t.Fatal(err)
	}
	if qty != 2 {
		t.Errorf("cart qty = %d, want 2 (1 + 1 merged)", qty)
	}
}
