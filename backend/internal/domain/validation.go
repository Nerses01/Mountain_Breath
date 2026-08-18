package domain

// Validation codes for the `fields` map of the error envelope.
//
// These are KEYS, not sentences. The API used to answer with English prose
// ("must be lowercase letters/digits separated by dashes"), which quietly
// baked one language into the contract: a Russian-speaking customer got a
// Russian page with an English validation error under the input. The client
// now looks the code up in its own message catalogue, so the same response
// renders correctly in all three languages — and a fourth needs no backend
// change at all.
//
// Constants rather than bare strings so a typo is a compile error and the
// whole vocabulary is greppable from one place. They are part of the public
// API contract: renaming one is a breaking change, exactly like renaming a
// JSON field.
const (
	ValidationRequired         = "required"
	ValidationSlugFormat       = "slug_format"
	ValidationEmailFormat      = "email_format"
	ValidationPasswordTooShort = "password_too_short"
	// A5: the change-password form's "current password is wrong" — a FIELD
	// error on current_password, not a 401: the session is fine, one input
	// is not.
	ValidationIncorrectPassword = "incorrect_password"
	ValidationPositive         = "positive"
	ValidationNotNegative      = "not_negative"
	ValidationVariantsRequired = "variants_required"
	// A gallery must nominate exactly one hero. The database enforces "at
	// most one" with a partial unique index; this catches "none" and turns
	// "two" into a field error instead of a constraint violation.
	ValidationOnePrimary = "one_primary_image"

	// E4. ValidationTooLong carries no number: the limit belongs to the
	// backend, but the sentence belongs to the frontend, and interpolating
	// "4000" into a code would put one language's punctuation into the
	// contract. The client renders "Keep it under {{max}} characters" from
	// its own catalogue.
	ValidationRatingRange   = "rating_range"
	ValidationTooLong       = "too_long"
	ValidationInvalidStatus = "invalid_status"

	// E5. A prices map keyed by a currency the shop does not sell in — the
	// same class of error as an unsupported locale, and caught the same way,
	// before the foreign key on variant_prices can turn it into a 500.
	ValidationUnknownCurrency = "unknown_currency"

	// E6. Codes, not sentences, as always — the second one exists because
	// "cash on delivery is AMD-only" is a rule the customer can FIX (switch
	// currency, or pick another method), so the message the client renders
	// for it must say more than "invalid".
	ValidationInvalidPaymentMethod = "invalid_payment_method"
	ValidationCashIsAMDOnly        = "cash_is_amd_only"

	// E7. Why a promo code cannot apply, in the order Promo.Issue checks
	// them. "Unknown" deliberately covers inactive and not-yet-started codes
	// too: confirming that a disabled or unannounced code exists is exactly
	// what a code-guesser wants to learn (the 404-not-403 reasoning again).
	// The rest are specific because the customer can ACT on them — wait,
	// spend more, switch market, pick another code.
	ValidationPromoUnknown     = "promo_unknown"
	ValidationPromoExpired     = "promo_expired"
	ValidationPromoUsed        = "promo_used"
	ValidationPromoExhausted   = "promo_exhausted"
	ValidationPromoNotInMarket = "promo_not_in_market"
	ValidationPromoMinSubtotal = "promo_min_subtotal"

	// A translations map keyed by a language the shop does not serve. Caught
	// here rather than at the database, whose CHECK constraint would answer
	// with a 500-shaped driver error instead of a field-level 400.
	ValidationLocaleUnsupported = "locale_unsupported"
	// English belongs in the `name`/`description` fields, which are the
	// fallback every other locale resolves to. Accepting it in `translations`
	// as well would mean two places to write one value, and no rule for which
	// wins when they disagree.
	ValidationLocaleIsDefault = "locale_is_default"
)

// ValidateTranslationLocales checks the keys of a translations map. Values
// are validated by the caller, which knows what fields a translation holds.
//
// `prefix` is the JSON path the client used ("translations"), so the field
// keys come back as `translations.hy` — the same convention as
// `variants[0].sku`, which lets a form attach the error to the right input.
func ValidateTranslationLocales[T any](prefix string, translations map[Locale]T) map[string]string {
	fields := make(map[string]string)
	for locale := range translations {
		if _, ok := ParseLocale(string(locale)); !ok {
			fields[prefix+"."+string(locale)] = ValidationLocaleUnsupported
			continue
		}
		if locale == DefaultLocale {
			fields[prefix+"."+string(locale)] = ValidationLocaleIsDefault
		}
	}
	return fields
}

// PasswordMinLength is the rule ValidationPasswordTooShort refers to.
// Exported because the number belongs to the backend but the sentence
// belongs to the frontend — keeping it here means there is still one
// authority for the value, even though two layers mention it.
const PasswordMinLength = 8
