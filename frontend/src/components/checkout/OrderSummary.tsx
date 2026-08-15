import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { Cart, Preview } from '../../api/types'
import { formatMoney } from '../../lib/format'
import { secondaryCurrency } from '../../lib/currencies'
import { Price } from '../ui/Price'

/**
 * The design's dark summary card — the right column of both the cart and the
 * checkout screens. ONE component on purpose (the plan's own requirement):
 * the number a customer reads on the cart page and the number beside "Place
 * the order" must be the same rendering of the same server-computed figures,
 * so there is exactly one place a discrepancy could hide, and it is this
 * file's props.
 *
 * E7 split its inputs: the LINES come off the cart (what is in the basket),
 * every FIGURE comes off the preview (what it costs — which since E7 depends
 * on who is asking and what code they hold, so /cart no longer carries it).
 * Nothing is computed here; the one calculator lives in the Go domain layer
 * and this component draws its answer.
 */
export function OrderSummary({
  cart,
  preview,
  action,
}: {
  cart: Cart
  preview: Preview
  action?: ReactNode
}) {
  const { t } = useTranslation()

  const shippingValue = () => {
    if (preview.shipping_minor > 0) {
      return formatMoney(preview.shipping_minor, preview.currency)
    }
    // Why it is free deserves its name: the first-order perk is a promise
    // kept, not a coincidence of thresholds.
    return preview.first_delivery_free
      ? t('checkout:summary.firstDeliveryFree')
      : t('checkout:summary.freeShipping')
  }

  return (
    <section
      aria-label={t('checkout:summary.label')}
      className="flex flex-col gap-4 rounded-2xl bg-bark p-7"
    >
      <h2 className="font-display text-lg font-bold text-ink-on-dark">
        {t('common:itemCount', { count: cart.items.reduce((n, it) => n + it.qty, 0) })}
      </h2>

      <ul className="flex flex-col gap-3.5">
        {cart.items.map((it) => (
          <li key={it.variant_id} className="flex items-center gap-3.5">
            <div
              aria-hidden="true"
              className="size-14 shrink-0 rounded-xl bg-[repeating-linear-gradient(135deg,rgba(255,248,238,0.24)_0_7px,rgba(255,248,238,0.1)_7px_14px)]"
            />
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <span className="truncate font-display text-[0.9375rem] font-semibold text-ink-on-dark">
                {it.product_name}
              </span>
              <span className="text-xs text-ink-on-dark-soft">
                {it.label} · ×{it.qty}
              </span>
            </div>
            <span className="font-display text-[0.9375rem] font-bold text-honey">
              {formatMoney(it.line_total_minor, cart.currency)}
            </span>
          </li>
        ))}
      </ul>

      <dl className="flex flex-col gap-2.5 border-t border-bark-soft pt-4">
        <SummaryRow
          label={t('checkout:summary.subtotal')}
          value={formatMoney(preview.subtotal_minor, preview.currency)}
        />
        <SummaryRow
          // The fee is named for what it is: a chilled parcel costs more,
          // and the label saying so is the design's own habit.
          label={
            preview.has_cold_chain
              ? t('checkout:summary.chilledShipping')
              : t('checkout:summary.shipping')
          }
          value={shippingValue()}
        />
        {/* The two discount lines the mock draws, each only when it earned
            its place — a permanent "− $0.00" row would be noise. Honey
            accent on the amounts, as the design colours them. */}
        {preview.member_discount_minor > 0 && (
          <SummaryRow
            label={t('checkout:summary.memberDiscount')}
            value={`− ${formatMoney(preview.member_discount_minor, preview.currency)}`}
            accent
          />
        )}
        {preview.promo_discount_minor > 0 && preview.promo_code && (
          <SummaryRow
            label={t('checkout:summary.promo', { code: preview.promo_code })}
            value={`− ${formatMoney(preview.promo_discount_minor, preview.currency)}`}
            accent
          />
        )}
      </dl>

      <div className="flex items-end justify-between border-t border-bark-soft pt-4">
        <span className="font-display text-[1.0625rem] font-bold text-ink-on-dark">
          {t('checkout:summary.total')}
        </span>
        <Price
          prices={hasBothMarkets(preview) ? preview.totals : undefined}
          primaryMinor={preview.total_minor}
          size="lg"
          tone="on-dark"
          className="items-end"
        />
      </div>

      {action}
    </section>
  )
}

function SummaryRow({
  label,
  value,
  accent,
}: {
  label: string
  value: string
  accent?: boolean
}) {
  return (
    <div className="flex justify-between text-[0.9375rem]">
      <dt className="text-ink-on-dark-body">{label}</dt>
      <dd className={accent ? 'text-honey' : 'text-ink-on-dark'}>{value}</dd>
    </div>
  )
}

// The muted second total only earns its line when the other market is
// actually priced — a basket the shop cannot quote in drams (or whose promo
// does not exist there) shows dollars alone rather than a dash.
function hasBothMarkets(preview: Preview): boolean {
  return secondaryCurrency(preview.currency, preview.totals) !== undefined
}
