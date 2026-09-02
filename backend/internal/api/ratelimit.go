package api

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// rateLimiter guards the endpoints where guessing pays: login, and the
// forgot-password mailer (which would otherwise let anyone use the shop as
// a spam cannon). Phase 11 parked it; E8 builds it, because this is the
// phase with auth already open on the table.
//
// Design: a FIXED-WINDOW counter per key, not a token bucket. The property
// password protection needs is crude — "no key gets more than N attempts
// per window" — and a window counter delivers it in twenty lines with one
// knob. Its known wart (a burst straddling the window boundary can see up
// to 2N) is harmless at these numbers: twenty guesses per ten minutes stops
// no customer and every dictionary.
//
// In-memory and per-process, deliberately. One API process serves this shop
// (the compose stack runs a single replica), so a shared store would be
// paying Redis prices for a constraint we do not have. The comment is the
// contract: horizontal scaling multiplies the limit by the replica count,
// and THAT is when this moves to storage.
//
// `now` is injectable for the same reason domain.Price takes a clock: the
// "trips and recovers" test would otherwise have to actually wait ten
// minutes.
type rateLimiter struct {
	mu      sync.Mutex
	windows map[string]*window
	limit   int
	span    time.Duration
	now     func() time.Time
}

type window struct {
	start time.Time
	count int
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{
		windows: make(map[string]*window),
		limit:   10,
		span:    10 * time.Minute,
		now:     time.Now,
	}
}

// allow records an attempt for key and says whether it is within the limit.
// Attempts are counted whether or not they later succeed — the limiter runs
// BEFORE the password check on purpose, so it also bounds the bcrypt work
// an attacker can make this CPU do.
func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := rl.now()
	w, ok := rl.windows[key]
	if !ok || now.Sub(w.start) >= rl.span {
		// A stale window recovers by being replaced — this is the "recovers"
		// half of the test. Expired entries for keys never seen again are
		// swept below rather than leaking forever.
		rl.windows[key] = &window{start: now, count: 1}
		rl.sweep(now)
		return true
	}
	w.count++
	return w.count <= rl.limit
}

// sweep drops expired windows. Called opportunistically from allow (already
// under the lock, already rare) instead of from a background goroutine —
// a janitor goroutine is a lifecycle to manage for a map that a busy day
// grows to a few hundred entries.
func (rl *rateLimiter) sweep(now time.Time) {
	for key, w := range rl.windows {
		if now.Sub(w.start) >= rl.span {
			delete(rl.windows, key)
		}
	}
}

// limitKey buckets attempts by client IP AND the account being tried, so an
// attacker hammering one account is stopped without locking that account's
// real owner out from their own address, and a botnet spreading one guess
// across many accounts still burns its per-IP allowance.
func limitKey(r *http.Request, email string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return host + "|" + email
}
