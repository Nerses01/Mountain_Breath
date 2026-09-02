package domain

// View is how a storefront read should be rendered: in which language, and
// for which market. Both change what the store SELECTs — the locale picks a
// translation row, the currency picks a price row — so neither is a
// presentation detail the API layer could keep to itself.
//
// It exists because after E5 those two values travel together EVERYWHERE:
// the product page, the related panel, the cart, the catalog. Five
// signatures reading (ctx, slug, locale, currency) is the classic sign that
// two parameters are really one concept — Fowler calls the fix Introduce
// Parameter Object; in C++ it is the same instinct as passing one small
// `const Options&` instead of four positional arguments that can be swapped
// by mistake. Here it also means E6 can add a tax region without editing
// every signature again.
//
// A value type, copied freely: two strings, no pointers, no ownership.
type View struct {
	Locale   Locale
	Currency Currency
}

// EffectiveLocale and EffectiveCurrency apply the same "zero value means no
// opinion, not invalid" rule as ProductFilter, so a test that builds a bare
// View still reads English prices in dollars instead of matching no rows.
func (v View) EffectiveLocale() Locale {
	if v.Locale == "" {
		return DefaultLocale
	}
	return v.Locale
}

func (v View) EffectiveCurrency() Currency {
	if v.Currency == "" {
		return DefaultCurrency
	}
	return v.Currency
}

// View lets a filter hand its rendering half to the helpers that only need
// that much — attachVariants does not care about pagination or sort order.
func (f ProductFilter) View() View {
	return View{Locale: f.EffectiveLocale(), Currency: f.EffectiveCurrency()}
}
