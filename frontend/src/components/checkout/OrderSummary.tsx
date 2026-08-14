import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import type { Cart } from '../../api/types'
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
 * Every figure comes off the Cart response; nothing is computed here. Even
 * the subtotal is the server's — a component that summed line totals itself
 * would be a second implementation of arithmetic the backend already owns.
 *
 * `action` is the button slot: "Go to checkout" on the cart, "Place the
 * order" on the checkout. The discount row waits for E7 — rendering a
 * hardcoded "− $4.00" like the mock would be showing a promise the backend
 * cannot keep yet.
 */
export function OrderSummary({ cart, action }: { cart: Cart; action?: ReactNode }) {
  const { t } = useTranslation()

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
          value={formatMoney(cart.subtotal_minor, cart.currency)}
        />
        <SummaryRow
          // The fee is named for what it is: a chilled parcel costs more,
          // and the label saying so is the design's own habit.
          label={
            cart.has_cold_chain
              ? t('checkout:summary.chilledShipping')
              : t('checkout:summary.shipping')
          }
          value={
            cart.shipping_minor === 0
              ? t('checkout:summary.freeShipping')
              : formatMoney(cart.shipping_minor, cart.currency)
          }
        />
      </dl>

      <div className="flex items-end justify-between border-t border-bark-soft pt-4">
        <span className="font-display text-[1.0625rem] font-bold text-ink-on-dark">
          {t('checkout:summary.total')}
        </span>
        <Price
          prices={hasBothMarkets(cart) ? cart.totals : undefined}
          primaryMinor={cart.total_minor}
          size="lg"
          tone="on-dark"
          className="items-end"
        />
      </div>

      {action}
    </section>
  )
}

function SummaryRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between text-[0.9375rem]">
      <dt className="text-ink-on-dark-body">{label}</dt>
      <dd className="text-ink-on-dark">{value}</dd>
    </div>
  )
}

// The muted second total only earns its line when the other market is
// actually priced — a basket the shop cannot quote in drams shows dollars
// alone rather than a dash.
function hasBothMarkets(cart: Cart): boolean {
  return secondaryCurrency(cart.currency, cart.totals) !== undefined
}
