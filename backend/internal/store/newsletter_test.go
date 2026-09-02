package store_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/store"
)

// The whole double-opt-in lifecycle in one narrative: subscribe →
// unconfirmed; confirm → live; unsubscribe → out; resubscribe → a NEW
// consent that must be proven again.
func TestNewsletterLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test (needs Docker)")
	}
	resetDB(t)
	s := store.New(testPool)
	ctx := context.Background()
	const email = "reader@test.local"

	status := func(t *testing.T) (confirmed, unsubscribed bool) {
		t.Helper()
		if err := testPool.QueryRow(ctx, `
			SELECT confirmed_at IS NOT NULL, unsubscribed_at IS NOT NULL
			FROM newsletter_subscribers WHERE email = $1`,
			email).Scan(&confirmed, &unsubscribed); err != nil {
			t.Fatal(err)
		}
		return confirmed, unsubscribed
	}

	// Subscribe: a row exists, but typing an address proves nothing.
	needs, err := s.SubscribeNewsletter(ctx, email, "token-1")
	if err != nil {
		t.Fatal(err)
	}
	if !needs {
		t.Fatal("first subscribe should need confirmation")
	}
	if confirmed, _ := status(t); confirmed {
		t.Fatal("subscribed WITHOUT the inbox's say-so")
	}

	// The inbox owner clicks. Twice, because people do — both succeed.
	if err := s.ConfirmNewsletter(ctx, "token-1"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := s.ConfirmNewsletter(ctx, "token-1"); err != nil {
		t.Fatalf("re-confirm should be idempotent: %v", err)
	}
	if confirmed, _ := status(t); !confirmed {
		t.Fatal("not confirmed after the click")
	}

	// Re-subscribing while live sends NOTHING (no spam loop) and must not
	// rotate the token — the unsubscribe links already in sent mail point
	// at it.
	needs, err = s.SubscribeNewsletter(ctx, email, "token-2")
	if err != nil {
		t.Fatal(err)
	}
	if needs {
		t.Error("a live subscription re-asked for confirmation")
	}
	if err := s.UnsubscribeNewsletter(ctx, "token-1"); err != nil {
		t.Errorf("the ORIGINAL token stopped working after a re-subscribe: %v", err)
	}
	if _, unsubscribed := status(t); !unsubscribed {
		t.Fatal("not unsubscribed")
	}

	// Coming back is a new consent: needs confirmation again, on a fresh
	// token, and the row is not live until the new link is clicked.
	needs, err = s.SubscribeNewsletter(ctx, email, "token-3")
	if err != nil {
		t.Fatal(err)
	}
	if !needs {
		t.Fatal("a returning subscriber must confirm again")
	}
	if confirmed, _ := status(t); confirmed {
		t.Fatal("returning subscriber live before the new click")
	}
	if err := s.ConfirmNewsletter(ctx, "token-3"); err != nil {
		t.Fatal(err)
	}
	if confirmed, unsubscribed := status(t); !confirmed || unsubscribed {
		t.Fatalf("after the return: confirmed=%v unsubscribed=%v", confirmed, unsubscribed)
	}

	// An invented token is the only failure in the whole flow.
	if err := s.ConfirmNewsletter(ctx, "no-such-token"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("invented token: %v, want ErrNotFound", err)
	}
}
