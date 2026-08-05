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
)

// PasswordMinLength is the rule ValidationPasswordTooShort refers to.
// Exported because the number belongs to the backend but the sentence
// belongs to the frontend — keeping it here means there is still one
// authority for the value, even though two layers mention it.
const PasswordMinLength = 8
