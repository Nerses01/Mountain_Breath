/**
 * The three languages the shop speaks, and the one it defaults to.
 *
 * Everything else derives from this file — the i18n config, the router, the
 * language switcher and the `Accept-Language` header sent to the API. Adding
 * a fourth language should mean editing this list and adding message files,
 * not hunting for hardcoded 'en' | 'hy' | 'ru' unions.
 */
export const LOCALES = ['en', 'hy', 'ru'] as const

export type Locale = (typeof LOCALES)[number]

/** English is the default and, deliberately, carries no URL prefix. */
export const DEFAULT_LOCALE: Locale = 'en'

/**
 * `nativeName` is what the switcher shows: a Russian speaker looks for
 * "Русский", not "Russian". `htmlLang` feeds the <html lang> attribute,
 * which drives screen-reader pronunciation and browser translation offers.
 */
export const LOCALE_META: Record<Locale, { nativeName: string; htmlLang: string }> = {
  en: { nativeName: 'English', htmlLang: 'en' },
  hy: { nativeName: 'Հայերեն', htmlLang: 'hy' },
  ru: { nativeName: 'Русский', htmlLang: 'ru' },
}

export function isLocale(value: string | undefined): value is Locale {
  return value !== undefined && (LOCALES as readonly string[]).includes(value)
}

/**
 * The locale a path is in, from its first segment. Anything that is not a
 * known locale — `/cart`, `/de/x` — means English, since English is the
 * unprefixed default rather than an error case.
 */
export function localeFromPath(pathname: string): Locale {
  const first = pathname.split('/').filter(Boolean)[0]
  return isLocale(first) ? first : DEFAULT_LOCALE
}

/** Every locale except the default, i.e. the ones that carry a URL prefix. */
export const PREFIXED_LOCALES = LOCALES.filter((l) => l !== DEFAULT_LOCALE)

/**
 * The path prefix for a locale: '' for English, '/hy' and '/ru' otherwise.
 * Keeping this one function means the "English has no prefix" rule is stated
 * once rather than re-derived at every call site.
 */
export function localePrefix(locale: Locale): string {
  return locale === DEFAULT_LOCALE ? '' : `/${locale}`
}

/**
 * Rewrites a path into another locale, preserving everything after the
 * prefix — so switching language on /hy/products/honey lands you on
 * /ru/products/honey rather than dumping you at the home page.
 */
export function pathForLocale(pathname: string, next: Locale): string {
  const segments = pathname.split('/').filter(Boolean)
  if (isLocale(segments[0])) {
    segments.shift()
  }
  const rest = segments.join('/')
  const prefix = localePrefix(next)
  return `${prefix}/${rest}`.replace(/\/+$/, '') || '/'
}
