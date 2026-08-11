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
	ValidationPositive         = "positive"
	ValidationNotNegative      = "not_negative"
	ValidationVariantsRequired = "variants_required"
	// A gallery must nominate exactly one hero. The database enforces "at
	// most one" with a partial unique index; this catches "none" and turns
	// "two" into a field error instead of a constraint violation.
	ValidationOnePrimary = "one_primary_image"

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
