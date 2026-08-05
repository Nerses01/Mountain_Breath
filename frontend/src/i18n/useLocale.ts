import { useEffect } from 'react'
import { useLocation } from 'react-router'
import { useTranslation } from 'react-i18next'
import {
  DEFAULT_LOCALE,
  isLocale,
  LOCALE_META,
  localeFromPath,
  pathForLocale,
  type Locale,
} from './locales'

/**
 * The one way to ask "what language are we in?".
 *
 * Reads the locale from the URL, not from i18next — a shared link must open
 * in the language it was shared in. i18next is then synced to follow, which
 * is why this hook also owns the <html lang> attribute.
 *
 * It parses `useLocation().pathname` rather than reading a `:locale` route
 * param, because the routes enumerate real prefixes (`/hy`, `/ru`) instead
 * of using a wildcard param — a `/:locale` param would happily match `/cart`
 * and treat "cart" as a language. Parsing the path keeps this hook working
 * regardless of how the route tree is shaped.
 *
 * Every page should call this rather than reaching for `i18next.language`
 * directly, so there is one place to change if the scheme ever moves.
 */
export function useLocale(): {
  locale: Locale
  /** The same page in another language, as a path you can navigate to. */
  hrefFor: (next: Locale) => string
  /** Prefixes an app-absolute path with the active locale. */
  localePath: (path: string) => string
} {
  const { pathname } = useLocation()
  const { i18n } = useTranslation()

  const locale = localeFromPath(pathname)

  useEffect(() => {
    if (i18n.language !== locale) {
      void i18n.changeLanguage(locale)
    }
    // <html lang> is not decoration: it tells a screen reader which voice to
    // use, and the browser whether to offer translation.
    document.documentElement.lang = LOCALE_META[locale].htmlLang
  }, [locale, i18n])

  return {
    locale,
    hrefFor: (next: Locale) => pathForLocale(pathname, next),
    localePath: (path: string) => pathForLocale(path, locale),
  }
}

export { isLocale, DEFAULT_LOCALE }
export type { Locale }
