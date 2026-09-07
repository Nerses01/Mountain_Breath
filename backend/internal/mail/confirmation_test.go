package mail_test

import (
	"strings"
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
	"github.com/Nerses01/Mountain_Breath/backend/internal/mail"
)

// Decision #110: the confirmation mail tells the customer HOW to pay, built
// from the same domain pieces the order page draws (TransferReference,
// BankDetails) — pinned here so the two cannot drift apart. The copy itself
// is flagged for native review like every translation; what is pinned is
// which facts each method's paragraph carries.
func TestOrderConfirmation_HowToPay(t *testing.T) {
	order := func(method string) domain.Order {
		return domain.Order{
			ID: 42, Locale: domain.LocaleEN, Currency: domain.CurrencyAMD, TotalMinor: 6400,
			PaymentMethod: method, PaymentStatus: domain.PaymentUnpaid,
		}
	}
	// A local Armenian account number — the country is not in the IBAN
	// registry, and the field takes whatever the bank prints.
	const account = "1570001234567890"
	bank := domain.BankDetails{Recipient: "Mountain Breath", Bank: "Ameriabank", Account: account}

	t.Run("a transfer names the account and the purpose line", func(t *testing.T) {
		msg := mail.OrderConfirmation(domain.LocaleEN, "a@test.local", order(domain.PayBankTransfer), "https://x/orders/42", bank)
		for _, want := range []string{account, "Ameriabank", "Mountain Breath", "MB-42"} {
			if !strings.Contains(msg.Text, want) {
				t.Errorf("mail lacks %q:\n%s", want, msg.Text)
			}
		}
	})

	t.Run("an unconfigured account promises the details and prints no blanks", func(t *testing.T) {
		msg := mail.OrderConfirmation(domain.LocaleEN, "a@test.local", order(domain.PayBankTransfer), "u", domain.BankDetails{})
		if !strings.Contains(msg.Text, "email you the account details") || strings.Contains(msg.Text, "account  ") {
			t.Errorf("unexpected text:\n%s", msg.Text)
		}
	})

	t.Run("cash says the amount to have ready, and nothing about accounts", func(t *testing.T) {
		msg := mail.OrderConfirmation(domain.LocaleEN, "a@test.local", order(domain.PayCashOnDelivery), "u", bank)
		if !strings.Contains(msg.Text, "in cash") || strings.Contains(msg.Text, account) {
			t.Errorf("unexpected text:\n%s", msg.Text)
		}
	})

	t.Run("every locale carries the paragraph", func(t *testing.T) {
		for _, l := range []domain.Locale{domain.LocaleHY, domain.LocaleRU} {
			o := order(domain.PayBankTransfer)
			o.Locale = l
			msg := mail.OrderConfirmation(l, "a@test.local", o, "u", bank)
			if !strings.Contains(msg.Text, "MB-42") || !strings.Contains(msg.Text, account) {
				t.Errorf("%s: no how-to-pay in\n%s", l, msg.Text)
			}
		}
	})
}
