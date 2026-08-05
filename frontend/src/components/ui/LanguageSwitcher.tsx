import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { cx } from '../../lib/cx'
import { LOCALE_META, LOCALES } from '../../i18n/locales'
import { useLocale } from '../../i18n/useLocale'

/**
 * Language chooser.
 *
 * Rendered as LINKS, not buttons: each language is a real, distinct URL, so
 * a link gives middle-click, "open in new tab" and a right-click "copy link"
 * that all behave correctly — and the choice survives a page reload because
 * it lives in the address bar rather than in component state.
 *
 * `hrefFor` keeps you on the current page when switching, so changing
 * language from a product page lands on the same product, not the home page.
 *
 * Each language is written in its OWN script (Հայերեն, not "Armenian"): a
 * reader looking for their language cannot necessarily read the name of it
 * in English.
 */
export function LanguageSwitcher({ className }: { className?: string }) {
  const { locale, hrefFor } = useLocale()
  const { t } = useTranslation()

  return (
    <nav aria-label={t('common:language.label')} className={className}>
      <ul className="flex items-center gap-3">
        {LOCALES.map((code) => {
          const isCurrent = code === locale
          return (
            <li key={code}>
              <Link
                to={hrefFor(code)}
                // `lang` on the link itself so a screen reader pronounces
                // "Русский" with a Russian voice rather than mangling it
                // with the surrounding page's language.
                lang={LOCALE_META[code].htmlLang}
                aria-current={isCurrent ? 'true' : undefined}
                className={cx(
                  'text-xs transition',
                  isCurrent
                    ? 'font-semibold text-honey'
                    : 'text-ink-on-dark-soft hover:text-ink-on-dark',
                )}
              >
                {LOCALE_META[code].nativeName}
              </Link>
            </li>
          )
        })}
      </ul>
    </nav>
  )
}
