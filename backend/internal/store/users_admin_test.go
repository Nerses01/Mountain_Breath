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

func setRole(t *testing.T, userID int64, role string) {
	t.Helper()
	if _, err := testPool.Exec(context.Background(),
		`UPDATE users SET role = $2 WHERE id = $1`, userID, role); err != nil {
		t.Fatalf("seeding role: %v", err)
	}
}

// F2 (decision #96): the role write path. The invariant under test is a
// COUNT ("at least one admin"), which no index or CHECK can hold — so the
// suite drives it the way the oversell test drives stock: including two
// demotions racing for the same last seat.
func TestUpdateUserRole(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	adminID := seedUser(t, "boss@test.local")
	setRole(t, adminID, domain.RoleAdmin)
	customerID := seedUser(t, "helper@test.local")

	t.Run("promote, then the freed-up demote", func(t *testing.T) {
		u, err := s.UpdateUserRole(ctx, customerID, domain.RoleAdmin)
		if err != nil {
			t.Fatalf("promoting: %v", err)
		}
		if u.Role != domain.RoleAdmin {
			t.Errorf("role = %q, want admin", u.Role)
		}
		// Two admins now — demoting the original is allowed.
		u, err = s.UpdateUserRole(ctx, adminID, domain.RoleCustomer)
		if err != nil {
			t.Fatalf("demoting with a second admin present: %v", err)
		}
		if u.Role != domain.RoleCustomer {
			t.Errorf("role = %q, want customer", u.Role)
		}
	})

	t.Run("the last admin cannot be demoted", func(t *testing.T) {
		// After the run above, helper is the only admin left.
		_, err := s.UpdateUserRole(ctx, customerID, domain.RoleCustomer)
		if !errors.Is(err, domain.ErrLastAdmin) {
			t.Errorf("err = %v, want ErrLastAdmin", err)
		}
		var count int
		if err := testPool.QueryRow(ctx,
			`SELECT count(*) FROM users WHERE role = 'admin'`).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("admin count = %d after refused demote, want 1", count)
		}
	})

	t.Run("unknown user is ErrNotFound", func(t *testing.T) {
		if _, err := s.UpdateUserRole(ctx, 99999, domain.RoleAdmin); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("err = %v, want ErrNotFound", err)
		}
	})
}

// The oversell test's shape applied to the admin seat: two admins, two
// concurrent demotions — exactly one may succeed, and the count must land
// on 1, never 0. Without the ordered FOR UPDATE on the admin set, both
// transactions would read "2 admins" and both demote.
func TestUpdateUserRole_ConcurrentDemotionsKeepOneAdmin(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	a := seedUser(t, "a@test.local")
	b := seedUser(t, "b@test.local")
	setRole(t, a, domain.RoleAdmin)
	setRole(t, b, domain.RoleAdmin)

	var succeeded, refused atomic.Int64
	var wg sync.WaitGroup
	for _, id := range []int64{a, b} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.UpdateUserRole(ctx, id, domain.RoleCustomer)
			switch {
			case err == nil:
				succeeded.Add(1)
			case errors.Is(err, domain.ErrLastAdmin):
				refused.Add(1)
			default:
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if succeeded.Load() != 1 || refused.Load() != 1 {
		t.Errorf("succeeded=%d refused=%d, want exactly 1 and 1",
			succeeded.Load(), refused.Load())
	}
	var count int
	if err := testPool.QueryRow(ctx,
		`SELECT count(*) FROM users WHERE role = 'admin'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("admin count = %d, want 1 — the invariant's whole point", count)
	}
}

func TestListUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()

	variantID := seedCatalog(t, 10)
	buyerID := seedUserWithCart(t, "buyer@test.local", variantID, 1)
	if _, err := s.CreateOrder(ctx, buyerID, domain.View{Currency: domain.CurrencyUSD}, testCheckout()); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	seedUser(t, "newest@test.local")

	users, counts, err := s.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 || len(counts) != 2 {
		t.Fatalf("got %d users / %d counts, want 2 / 2", len(users), len(counts))
	}
	// Newest first; the buyer's single order is counted, the newcomer's zero.
	if users[0].Email != "newest@test.local" || counts[0] != 0 {
		t.Errorf("first row = %s (%d orders), want the newcomer with 0", users[0].Email, counts[0])
	}
	if users[1].Email != "buyer@test.local" || counts[1] != 1 {
		t.Errorf("second row = %s (%d orders), want the buyer with 1", users[1].Email, counts[1])
	}
}
