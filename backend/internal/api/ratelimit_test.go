package api

import (
	"testing"
	"time"
)

// White-box on purpose (package api, not api_test): the limiter's clock is
// an unexported field, and injecting it through the public surface would
// mean shipping a test knob in Options. The "trips" half is also covered
// black-box through the login handler; THIS test owns "recovers", because
// recovery is ten minutes long on a real clock.
func TestRateLimiter(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	rl := newRateLimiter()
	rl.now = func() time.Time { return now }

	for i := 1; i <= rl.limit; i++ {
		if !rl.allow("ip|a@x") {
			t.Fatalf("attempt %d refused within the limit", i)
		}
	}
	if rl.allow("ip|a@x") {
		t.Error("attempt over the limit allowed")
	}

	// Another key is another budget — the neighbour is not locked out.
	if !rl.allow("ip|b@x") {
		t.Error("a different key shares the exhausted budget")
	}

	// The window passes; the same key recovers.
	now = now.Add(rl.span)
	if !rl.allow("ip|a@x") {
		t.Error("key did not recover after the window")
	}

	// The sweep dropped the expired stranger too, not just the caller's key.
	rl.mu.Lock()
	_, stillThere := rl.windows["ip|b@x"]
	rl.mu.Unlock()
	if stillThere {
		t.Error("expired window survived the sweep")
	}
}
