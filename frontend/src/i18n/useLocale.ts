import { useEffect } from 'react'
import { useLocation, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import {
  DEFAULT_LOCALE,
  isLocale,
  LOCALE_META,
  pathForLocale,
  type Locale,
} from './locales'

/**
 * The one way to ask "what language are we in?".
 *
 * Reads the locale from the ROUTE, not from i18next — the URL is the source
 * of truth, so a shared link always opens in the language it was shared in.
 * i18next is then synced to follow, which is why this hook is also where the
 * <html lang> attribute gets set.
 *
 * Every page should call this rather than reaching for `i18next.language`
 * directly, so there is exactly one place to change if the routing scheme
 * ever moves (a `?lang=` param, a subdomain).
 */
export function useLocale(): {
  locale: Locale
  /** The same page in another language, as a path you can navigate to. */
  hrefFor: (next: Locale) => string
} {
  const { locale: raw } = useParams()
  const { pathname } = useLocation()
  const { i18n } = useTranslation()

  // An unknown prefix (/de/...) is not an error — it simply is not a locale
  // segment, so it falls back to English and the router treats it as a
  // normal path.
  const locale: Locale = isLocale(raw) ? raw : DEFAULT_LOCALE

  useEffect(() => {
    if (i18n.language !== locale) {
      void i18n.changeLanguage(locale)
    }
    // <html lang> is not decoration: it tells a screen reader which voice to
    // use, and tells the browser whether to offer translation.
    document.documentElement.lang = LOCALE_META[locale].htmlLang
  }, [locale, i18n])

  return {
    locale,
    hrefFor: (next: Locale) => pathForLocale(pathname, next),
  }
}
