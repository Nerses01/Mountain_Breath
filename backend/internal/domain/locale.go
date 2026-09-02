package domain

// Locale is one of the languages the shop speaks. It lives in domain rather
// than in the API layer because it changes what the store SELECTs, not just
// how a response is rendered — both layers need the same vocabulary.
type Locale string

const (
	LocaleEN Locale = "en"
	LocaleHY Locale = "hy"
	LocaleRU Locale = "ru"

	// DefaultLocale is what an unrecognised or absent request resolves to.
	// English, matching the frontend's unprefixed routes.
	DefaultLocale = LocaleEN
)

// Locales is the whole supported set, in preference order. Mirrors
// frontend/src/i18n/locales.ts and the CHECK constraint in migration 000007
// — three places that must agree, which is the price of the constraint
// catching a bad locale at the database boundary.
var Locales = []Locale{LocaleEN, LocaleHY, LocaleRU}

// ParseLocale accepts a bare tag ("hy") or a full one ("hy-AM", "ru_RU"),
// since Accept-Language and browser settings send both. Reports whether the
// value named a language we actually serve — callers decide whether an
// unknown one is an error or simply falls back.
func ParseLocale(s string) (Locale, bool) {
	if s == "" {
		return DefaultLocale, false
	}
	// Cut any region or script subtag: "hy-AM" and "hy" are the same shop
	// language, and we do not localise per region.
	for i, r := range s {
		if r == '-' || r == '_' {
			s = s[:i]
			break
		}
	}
	s = lowerASCII(s)
	for _, l := range Locales {
		if string(l) == s {
			return l, true
		}
	}
	return DefaultLocale, false
}

// SearchConfig maps a locale to its Postgres text search configuration.
//
// The mapping is a whitelist, not string formatting: the result is
// interpolated into a query as a regconfig, so returning caller-controlled
// text here would be an injection vector. Every branch returns a constant.
//
// Postgres ships all three (SELECT cfgname FROM pg_ts_config lists 29), so
// every language the shop speaks gets real stemming rather than falling back
// to the `simple` config.
func (l Locale) SearchConfig() string {
	switch l {
	case LocaleHY:
		return "armenian"
	case LocaleRU:
		return "russian"
	default:
		return "english"
	}
}

func (l Locale) String() string { return string(l) }

// lowerASCII avoids strings.ToLower's Unicode machinery for what is only
// ever an ASCII language tag.
func lowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
