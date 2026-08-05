package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

type Category struct {
	ID        int64
	Slug      string
	SortOrder int
	CreatedAt time.Time

	// Name is the English name — the value every other locale falls back to,
	// and what reads return when no translation matches. On a read it holds
	// the RESOLVED name for the requested locale; on a write it is always
	// English. That dual meaning is deliberate: it keeps one field for "the
	// name to show" instead of forcing every caller to resolve one.
	Name string

	// Translations holds the non-default locales only. English lives in Name,
	// so a key of "en" here is a validation error rather than a second place
	// to write the same value.
	Translations map[Locale]string
}

// ErrSlugTaken is a sentinel error: the store returns it, the API layer
// translates it to 409 Conflict. Neither layer needs to know the other's
// details — they only share this value.
var ErrSlugTaken = errors.New("slug already taken")

// slugRe: lowercase words of letters/digits separated by single dashes,
// e.g. "herbal-tea", "coffee", "gift-sets-2026".
var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// ValidateCategory returns a field->code map; empty map means valid.
// The values are validation CODES the client translates, not sentences —
// see validation.go.
func ValidateCategory(slug, name string) map[string]string {
	fields := make(map[string]string)
	if strings.TrimSpace(name) == "" {
		fields["name"] = ValidationRequired
	}
	switch {
	case slug == "":
		fields["slug"] = ValidationRequired
	case !slugRe.MatchString(slug):
		fields["slug"] = ValidationSlugFormat
	}
	return fields
}

// ValidateCategoryTranslations checks the optional per-locale names.
//
// A blank translation is REJECTED rather than ignored: the form is allowed to
// leave a language out entirely (it falls back to English), so sending a key
// with an empty value means something went wrong rather than "no translation".
func ValidateCategoryTranslations(tr map[Locale]string) map[string]string {
	fields := ValidateTranslationLocales("translations", tr)
	for locale, name := range tr {
		if strings.TrimSpace(name) == "" {
			fields["translations."+string(locale)+".name"] = ValidationRequired
		}
	}
	return fields
}
