import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import { DEFAULT_LOCALE, LOCALES } from './locales'
import { en } from './messages/en'
import { hy } from './messages/hy'
import { ru } from './messages/ru'

/**
 * i18next setup.
 *
 * No language DETECTOR is wired in, deliberately. The URL is the single
 * source of truth for locale (`/hy/...`, `/ru/...`, bare `/` for English),
 * so a detector reading `navigator.language` or a cookie would be a second,
 * competing opinion — and the two disagreeing is how you get a page whose
 * URL says Armenian while its text says English. `useLocale` syncs i18next
 * from the route instead, one direction only.
 *
 * Messages are bundled rather than fetched: three small catalogues cost less
 * than the request that would load one, and there is no flash of untranslated
 * text on first paint.
 */
void i18n.use(initReactI18next).init({
  resources: { en, hy, ru },
  lng: DEFAULT_LOCALE,
  fallbackLng: DEFAULT_LOCALE,
  supportedLngs: LOCALES,
  defaultNS: 'common',
  // A missing Armenian key falls through to English rather than rendering
  // the raw key or an empty string — a partially translated page still
  // reads, which matters while translations are being filled in.
  fallbackNS: false,
  interpolation: {
    // React escapes interpolated values already; letting i18next escape them
    // too would double-encode apostrophes and ampersands.
    escapeValue: false,
  },
  react: {
    useSuspense: false,
  },
})

export default i18n
