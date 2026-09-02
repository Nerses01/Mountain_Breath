import { useTranslation } from 'react-i18next'
import { cx } from '../../lib/cx'
import { CURRENCIES, CURRENCY_META } from '../../lib/currencies'
import { useCurrency } from '../../lib/useCurrency'

/**
 * The footer's "USD / AMD", made real.
 *
 * Removed post-A5 (decision #90), restored by #98 — see LanguageSwitcher;
 * the same anonymous-visitor argument, stronger even: a currency has no URL
 * form, so a signed-out visitor had NO way to switch at all.
 *
 * BUTTONS, not links — the opposite of LanguageSwitcher, and the difference
 * is the point. A language is part of the URL, so each one is a real
 * destination and deserves an anchor; a currency is a preference applied to
 * the page you are already on, so there is nothing to navigate to and a link
 * would be a lie to anyone who middle-clicks it.
 *
 * A radiogroup rather than a row of toggles: the options are mutually
 * exclusive and exactly one is always chosen, which is what
 * `role="radio" + aria-checked` announces. Arrow keys are not implemented
 * because a native radiogroup's roving focus is only expected when the group
 * takes ONE tab stop; with two buttons in a footer bar, tabbing to each is
 * both simpler and unsurprising.
 */
export function CurrencySwitcher({ className }: { className?: string }) {
  const { currency, setCurrency } = useCurrency()
  const { t } = useTranslation()

  return (
    <div
      role="radiogroup"
      aria-label={t('common:currency.label')}
      className={cx('flex items-center gap-2', className)}
    >
      {CURRENCIES.map((code, i) => (
        <span key={code} className="flex items-center gap-2">
          {i > 0 && (
            // Decorative: the separator the mock draws between the two codes.
            // aria-hidden so a screen reader reads "USD, AMD", not "USD
            // slash AMD".
            <span aria-hidden="true" className="text-ink-on-dark-muted">
              /
            </span>
          )}
          <button
            type="button"
            role="radio"
            aria-checked={code === currency}
            onClick={() => setCurrency(code)}
            className={cx(
              'rounded-full text-xs transition',
              code === currency
                ? 'font-semibold text-honey'
                : 'text-ink-on-dark-soft hover:text-ink-on-dark',
            )}
          >
            {/* The code, plus its symbol for readers who recognise the glyph
                faster than the letters. */}
            {code} <span aria-hidden="true">{CURRENCY_META[code].symbol}</span>
          </button>
        </span>
      ))}
    </div>
  )
}
