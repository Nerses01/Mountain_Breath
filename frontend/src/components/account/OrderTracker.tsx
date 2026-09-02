import { useTranslation } from 'react-i18next'
import type { Order, OrderStatus } from '../../api/types'
import { cx } from '../../lib/cx'
import { useLocale } from '../../i18n/useLocale'
import { CheckIcon } from '../ui/icons'

/**
 * A2 (canvas 07, decision log #85): the order's journey as four steps —
 * Placed → Confirmed → Shipped → Delivered. These are the REAL state
 * machine's states renamed in copy; the canvas's courier stages
 * ("out for delivery") exist nowhere in the system and are not drawn.
 *
 * Dates come from the recorded timeline (order.events). An order older
 * than the events table carries only its backfilled `pending` event, so
 * later steps show their state without a date — honest position over
 * invented history.
 *
 * Cancelled is not a fifth step: it is the journey ending, drawn as a flat
 * band (a state the canvas never draws — ours to design).
 *
 * Semantics: an <ol>, because a tracker IS an ordered list of steps; the
 * current step carries aria-current="step" (the spec's value for exactly
 * this), so a screen reader hears position, not just four labels.
 */
const STEPS = ['pending', 'confirmed', 'shipped', 'delivered'] as const

export function OrderTracker({ order }: { order: Order }) {
  const { t } = useTranslation()
  const { locale } = useLocale()

  const stepDate = (status: OrderStatus): string | undefined => {
    const ev = order.events.find((e) => e.status === status)
    if (!ev) return undefined
    return new Date(ev.created_at).toLocaleDateString(locale, { day: 'numeric', month: 'short' })
  }

  if (order.status === 'cancelled') {
    const date = stepDate('cancelled')
    return (
      <p className="rounded-2xl bg-panel px-5 py-3.5 text-sm font-semibold text-ink-muted">
        {t('account:tracker.cancelledBand')}
        {date && ` · ${date}`}
      </p>
    )
  }

  const currentIndex = STEPS.indexOf(order.status as (typeof STEPS)[number])

  return (
    <ol className="flex items-start" aria-label={t('account:tracker.label')}>
      {STEPS.map((step, i) => {
        const done = i < currentIndex || order.status === 'delivered'
        const current = i === currentIndex && !done
        const date = stepDate(step)
        return (
          // Each <li> holds its step AND the bar leading to the next — the
          // bar belongs to the transition, so it takes the DONE styling of
          // the step it leaves.
          <li
            key={step}
            aria-current={current ? 'step' : undefined}
            className={cx('flex items-start', i < STEPS.length - 1 && 'flex-1')}
          >
            <div className="flex w-16 flex-col items-center gap-1.5 sm:w-24">
              <span
                aria-hidden
                className={cx(
                  'flex size-7 items-center justify-center rounded-full text-xs font-bold sm:size-8',
                  done && 'bg-brand-ink text-ink-on-dark',
                  current && 'border-[2.5px] border-brand-ink bg-card text-brand-ink',
                  !done && !current && 'border-2 border-line bg-card text-ink-faint',
                )}
              >
                {done ? <CheckIcon size={15} /> : current ? '…' : i + 1}
              </span>
              <span
                className={cx(
                  'text-center text-xs leading-tight',
                  done || current ? 'font-bold text-ink' : 'font-semibold text-ink-muted',
                )}
              >
                {t(`account:tracker.${step}`)}
              </span>
              {/* The date line always renders (— when unknown) so the four
                  columns stay the same height and the bars stay level. */}
              <span className="text-2xs text-ink-muted">{date ?? '—'}</span>
            </div>
            {i < STEPS.length - 1 && (
              <span
                aria-hidden
                className={cx(
                  'mt-3.5 h-0.75 flex-1 rounded-full sm:mt-4',
                  i < currentIndex || order.status === 'delivered' ? 'bg-brand-ink' : 'bg-line',
                )}
              />
            )}
          </li>
        )
      })}
    </ol>
  )
}
