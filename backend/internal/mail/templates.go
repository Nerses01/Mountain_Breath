package mail

import (
	"fmt"
	"strings"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

// The two transactional messages E8 sends, in the shop's three languages.
//
// This copy lives in the BACKEND, unlike every UI string, because an email
// is rendered on the server — there is no browser catalogue between this
// text and the reader. The locale is whatever the request that triggered
// the mail had negotiated: a reset requested from /hy/ arrives in Armenian.
// Armenian and Russian are machine-assisted and flagged for native review,
// like the frontend catalogues before them.

type resetCopy struct {
	subject string
	body    string // %s = the reset link
	expiry  string
	ignore  string
}

var resetCopies = map[domain.Locale]resetCopy{
	domain.LocaleEN: {
		subject: "Reset your Mountain Breath password",
		body:    "Someone (hopefully you) asked to reset the password for this address.\n\nSet a new one here:\n%s",
		expiry:  "The link works once and expires in 1 hour.",
		ignore:  "If this wasn't you, ignore this email — your password is unchanged.",
	},
	domain.LocaleHY: {
		subject: "Վերականգնեք ձեր Mountain Breath գաղտնաբառը",
		body:    "Ինչ-որ մեկը (հուսով ենք՝ դուք) խնդրել է վերականգնել այս հասցեի գաղտնաբառը։\n\nՆոր գաղտնաբառ սահմանեք այստեղ․\n%s",
		expiry:  "Հղումը գործում է մեկ անգամ և ուժի մեջ է 1 ժամ։",
		ignore:  "Եթե դա դուք չէիք, անտեսեք այս նամակը — ձեր գաղտնաբառը չի փոխվել։",
	},
	domain.LocaleRU: {
		subject: "Сброс пароля Mountain Breath",
		body:    "Кто-то (надеемся, вы) запросил сброс пароля для этого адреса.\n\nЗадайте новый пароль здесь:\n%s",
		expiry:  "Ссылка работает один раз и действует 1 час.",
		ignore:  "Если это были не вы, просто игнорируйте письмо — пароль не изменён.",
	},
}

// ResetMessage builds the password-reset mail around the one-time link.
func ResetMessage(locale domain.Locale, to, link string) Message {
	c, ok := resetCopies[locale]
	if !ok {
		c = resetCopies[domain.LocaleEN]
	}
	return Message{
		To:      to,
		Subject: c.subject,
		Text:    fmt.Sprintf(c.body, link) + "\n\n" + c.expiry + "\n" + c.ignore + "\n",
	}
}

type orderCopy struct {
	subject string // %d = order id
	intro   string
	total   string // %s = formatted total
	link    string // %s = the order URL
}

var orderCopies = map[domain.Locale]orderCopy{
	domain.LocaleEN: {
		subject: "Order #%d — the hive is packing it",
		intro:   "Thank you! Your order is in:",
		total:   "Total: %s",
		link:    "The full receipt, and the parcel's progress:\n%s",
	},
	domain.LocaleHY: {
		subject: "Պատվեր #%d — փեթակն արդեն փաթեթավորում է",
		intro:   "Շնորհակալություն։ Ձեր պատվերն ընդունված է․",
		total:   "Ընդամենը՝ %s",
		link:    "Ամբողջական անդորրագիրը և ծանրոցի ընթացքը․\n%s",
	},
	domain.LocaleRU: {
		subject: "Заказ #%d — улей уже собирает посылку",
		intro:   "Спасибо! Ваш заказ принят:",
		total:   "Итого: %s",
		link:    "Полный чек и путь посылки:\n%s",
	},
}

type newsletterCopy struct {
	subject string
	body    string // %s = the confirm link
	ignore  string
}

var newsletterCopies = map[domain.Locale]newsletterCopy{
	domain.LocaleEN: {
		subject: "Confirm your harvest notes",
		body:    "One click and you're in: what is flowering, what we are jarring, once a month.\n\nConfirm here:\n%s",
		ignore:  "If you didn't ask for this, ignore it — nothing will be sent.",
	},
	domain.LocaleHY: {
		subject: "Հաստատեք բերքի նամակների բաժանորդագրությունը",
		body:    "Մեկ սեղմում, և պատրաստ է. ինչ է ծաղկում, ինչ ենք լցնում բանկաների մեջ՝ ամիսը մեկ։\n\nՀաստատեք այստեղ․\n%s",
		ignore:  "Եթե դուք չեք խնդրել սա, անտեսեք նամակը — ոչինչ չի ուղարկվի։",
	},
	domain.LocaleRU: {
		subject: "Подтвердите подписку на письма с пасеки",
		body:    "Один клик — и готово: что цветёт, что разливаем по банкам, раз в месяц.\n\nПодтвердите здесь:\n%s",
		ignore:  "Если вы этого не просили, просто игнорируйте письмо — ничего не придёт.",
	},
}

// NewsletterConfirmMessage is the double-opt-in mail: the click, not the
// form submit, is what consent means.
func NewsletterConfirmMessage(locale domain.Locale, to, link string) Message {
	c, ok := newsletterCopies[locale]
	if !ok {
		c = newsletterCopies[domain.LocaleEN]
	}
	return Message{
		To:      to,
		Subject: c.subject,
		Text:    fmt.Sprintf(c.body, link) + "\n\n" + c.ignore + "\n",
	}
}

// OrderConfirmation builds the receipt mail from an order's SNAPSHOTS — the
// same rule as the order page: names and prices as charged, one currency,
// no re-resolution against today's catalog.
func OrderConfirmation(locale domain.Locale, to string, o domain.Order, orderURL string) Message {
	c, ok := orderCopies[locale]
	if !ok {
		c = orderCopies[domain.LocaleEN]
	}

	var b strings.Builder
	b.WriteString(c.intro + "\n\n")
	for _, it := range o.Items {
		fmt.Fprintf(&b, "  %d × %s (%s) — %s\n",
			it.Qty, it.Name, it.Label, domain.FormatMinor(it.PriceMinor*int64(it.Qty), o.Currency))
	}
	b.WriteString("\n" + fmt.Sprintf(c.total, domain.FormatMinor(o.TotalMinor, o.Currency)) + "\n\n")
	b.WriteString(fmt.Sprintf(c.link, orderURL) + "\n")

	return Message{
		To:      to,
		Subject: fmt.Sprintf(c.subject, o.ID),
		Text:    b.String(),
	}
}
