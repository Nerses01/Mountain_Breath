package mail_test

import (
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/mail"
)

// F2: the status-mail builder. What is worth pinning is the SELECTION
// logic — which language, which status, and when nothing is sent — not the
// copy itself, which is flagged for native review like every translation.
func TestOrderStatusUpdate(t *testing.T) {
	order := func(locale domain.Locale, status string) domain.Order {
		return domain.Order{ID: 42, Locale: locale, Status: status}
	}

	t.Run("the order's language wins, not anyone's request", func(t *testing.T) {
		msg, ok := mail.OrderStatusUpdate("a@test.local", order(domain.LocaleHY, domain.OrderShipped), "https://x/orders/42")
		if !ok {
			t.Fatal("no message for a legal status")
		}
		if !strings.Contains(msg.Subject, "#42") || !strings.Contains(msg.Subject, "Պատվեր") {
			t.Errorf("subject = %q, want the Armenian shipped subject with #42", msg.Subject)
		}
		if !strings.Contains(msg.Text, "https://x/orders/42") {
			t.Errorf("body lacks the order URL: %q", msg.Text)
		}
	})

	t.Run("an unknown locale falls back to English", func(t *testing.T) {
		msg, ok := mail.OrderStatusUpdate("a@test.local", order("de", domain.OrderConfirmed), "u")
		if !ok || !strings.Contains(msg.Subject, "confirmed") {
			t.Errorf("ok=%v subject=%q, want the English confirmed subject", ok, msg.Subject)
		}
	})

	t.Run("each status gets its own subject", func(t *testing.T) {
		seen := map[string]bool{}
		for _, st := range []string{domain.OrderConfirmed, domain.OrderShipped, domain.OrderDelivered, domain.OrderCancelled} {
			msg, ok := mail.OrderStatusUpdate("a@test.local", order(domain.LocaleEN, st), "u")
			if !ok {
				t.Fatalf("no copy for %q", st)
			}
			if seen[msg.Subject] {
				t.Errorf("subject %q reused across statuses", msg.Subject)
			}
			seen[msg.Subject] = true
		}
	})

	t.Run("pending has no letter — the confirmation mail is that step", func(t *testing.T) {
		if _, ok := mail.OrderStatusUpdate("a@test.local", order(domain.LocaleEN, domain.OrderPending), "u"); ok {
			t.Error("a mail was built for pending")
		}
	})
}
