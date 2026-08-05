package domain_test

import (
	"testing"

	"github.com/Nerses01/Mountain_Breath/backend/internal/domain"
)

func TestParseLocale(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   domain.Locale
		wantOK bool
	}{
		{"bare tag", "hy", domain.LocaleHY, true},
		{"region subtag is cut", "hy-AM", domain.LocaleHY, true},
		{"underscore form", "ru_RU", domain.LocaleRU, true},
		{"uppercase", "RU", domain.LocaleRU, true},
		{"english", "en", domain.LocaleEN, true},
		// Unsupported input is not an error to the caller — it reports
		// false and hands back the default, so a shop never refuses to
		// render a page over a language tag.
		{"unsupported language", "de", domain.DefaultLocale, false},
		{"empty", "", domain.DefaultLocale, false},
		{"garbage", "!!", domain.DefaultLocale, false},
		// "hyena" must not match "hy" by prefix.
		{"longer word starting with a tag", "hyena", domain.DefaultLocale, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := domain.ParseLocale(tc.input)
			if got != tc.want || ok != tc.wantOK {
				t.Errorf("ParseLocale(%q) = %q, %v; want %q, %v",
					tc.input, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestLocaleSearchConfig(t *testing.T) {
	// These three strings are interpolated into SQL as a regconfig, so the
	// mapping being a closed whitelist is a security property, not a
	// convenience.
	tests := map[domain.Locale]string{
		domain.LocaleEN: "english",
		domain.LocaleHY: "armenian",
		domain.LocaleRU: "russian",
		"":              "english", // zero value must not produce ""::regconfig
		"de":            "english", // never reached in practice; must still be safe
	}

	for locale, want := range tests {
		if got := locale.SearchConfig(); got != want {
			t.Errorf("Locale(%q).SearchConfig() = %q; want %q", locale, got, want)
		}
	}
}

func TestProductFilterEffectiveLocale(t *testing.T) {
	// A zero-value filter must mean English, not the empty string — an empty
	// locale would match no translation row and blank every product name.
	var zero domain.ProductFilter
	if got := zero.EffectiveLocale(); got != domain.LocaleEN {
		t.Errorf("zero filter locale = %q; want %q", got, domain.LocaleEN)
	}

	set := domain.ProductFilter{Locale: domain.LocaleRU}
	if got := set.EffectiveLocale(); got != domain.LocaleRU {
		t.Errorf("explicit locale = %q; want %q", got, domain.LocaleRU)
	}
}
