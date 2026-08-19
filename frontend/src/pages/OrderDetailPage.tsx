import { useEffect, useRef, useState } from 'react'
import { Link, useLocation, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { ApiError } from '../api/client'
import { useCancelOrder, useOrder, useReorder } from '../api/hooks'
import { OrderTracker } from '../components/account/OrderTracker'
import { ReorderReport } from '../components/account/ReorderReport'
import { Button } from '../components/ui'
import { useLocale } from '../i18n/useLocale'
import { cx } from '../lib/cx'
import { formatMoney } from '../lib/format'

/**
 * /orders/:id — the confirmation page ("3 Done") and, later, the same order
 * reached from history. One page for both because they show the same record;
 * the only difference is the success banner, and THAT comes from router
 * state (`{ placed: true }` set by the checkout's navigate) rather than from
 * the URL — so a refresh or a shared link shows the order without
 * re-announcing "thank you", which was true once, at the moment of placing.
 *
 * Everything here renders SNAPSHOTS. No `Price` component, no dual currency,
 * no re-resolved product names: an order shows what was charged, in the one
 * currency it was charged in, addressed where it was sent — even if every
 * price and translation has changed since. The cart shows the live world;
 * this page shows a receipt.
 */
export function OrderDetailPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { id } = useParams()
  const location = useLocation()
  const order = useOrder(Number(id))

  const justPlaced = Boolean((location.state as { placed?: boolean } | null)?.placed)

  // No signed-in guard here anymore (A1): AccountLayout renders this pane
  // only for a signed-in user.
  if (order.isPending) {
    return <Shell>{t('common:state.loading')}</Shell>
  }
  if (order.isError || !order.data) {
    return (
      <Shell>
        <p className="text-ink-body">{t('order:notFound')}</p>
      </Shell>
    )
  }

  const o = order.data

  return (
    <Shell>
      {justPlaced && (
        // role="status": announced politely by screen readers on arrival.
        <div
          role="status"
          className="mb-6 flex items-start gap-3 rounded-2xl bg-honey/25 p-5"
        >
          <span
            aria-hidden="true"
            className="flex size-6 shrink-0 items-center justify-center rounded-full bg-honey text-xs font-bold text-ink"
          >
            ✓
          </span>
          <div>
            <p className="font-display font-bold text-ink">{t('order:placedTitle')}</p>
            <p className="text-sm text-ink-body">{t('order:placedBlurb')}</p>
          </div>
        </div>
      )}

      <div className="flex flex-wrap items-baseline justify-between gap-2">
        <h1 className="font-display text-display-sm font-extrabold text-ink">
          {t('account:orderNumber', { id: o.id })}
        </h1>
        <span className="text-sm text-ink-soft">
          {new Date(o.created_at).toLocaleDateString()} ·{' '}
          {t(`account:status.${o.status}`)}
        </span>
      </div>

      {/* A2: the same tracker as the orders screen — the detail page is
          the "Details" destination and must not look like another site. */}
      <div className="mt-6 rounded-2xl bg-card p-6">
        <OrderTracker order={o} />
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-[1fr_360px]">
        {/* ── The items, as charged ─────────────────────────────────── */}
        <section className="rounded-2xl bg-card p-6">
          <ul className="flex flex-col divide-y divide-line-soft">
            {o.items.map((it, i) => (
              <li key={i} className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
                <div className="flex flex-col">
                  <span className="font-medium text-ink">{it.name}</span>
                  <span className="text-xs text-ink-soft">
                    {it.label} · ×{it.qty}
                  </span>
                </div>
                <span className="font-display font-bold text-ink">
                  {formatMoney(it.price_minor * it.qty, o.currency)}
                </span>
              </li>
            ))}
          </ul>

          <dl className="mt-4 flex flex-col gap-2 border-t border-line pt-4 text-sm">
            <Row label={t('checkout:summary.subtotal')} value={formatMoney(o.subtotal_minor, o.currency)} />
            <Row
              label={t('checkout:summary.shipping')}
              value={o.shipping_minor === 0
                ? t('checkout:summary.freeShipping')
                : formatMoney(o.shipping_minor, o.currency)}
            />
            {/* E7: the split, drawn as the two lines the design names — and
                a generic "Discount" only for pre-split orders, whose lump
                sum cannot honestly claim either label. */}
            {o.member_discount_minor > 0 && (
              <Row
                label={t('order:memberDiscount')}
                value={'− ' + formatMoney(o.member_discount_minor, o.currency)}
              />
            )}
            {o.promo_discount_minor > 0 && o.promo_code && (
              <Row
                label={t('order:promo', { code: o.promo_code })}
                value={'− ' + formatMoney(o.promo_discount_minor, o.currency)}
              />
            )}
            {o.discount_minor > 0 &&
              o.member_discount_minor + o.promo_discount_minor === 0 && (
                <Row label={t('order:discount')} value={'− ' + formatMoney(o.discount_minor, o.currency)} />
              )}
            {/* Contained, not added: the line reads "includes VAT", and the
                figure takes no part in the sum below it. */}
            <Row label={t('order:includesVat')} value={formatMoney(o.tax_minor, o.currency)} muted />
          </dl>

          <div className="mt-4 flex items-baseline justify-between border-t border-line pt-4">
            <span className="font-display text-base font-bold text-ink">
              {t('checkout:summary.total')}
            </span>
            <span className="font-display text-2xl font-extrabold text-brand-ink">
              {formatMoney(o.total_minor, o.currency)}
            </span>
          </div>
        </section>

        {/* ── Where and how ─────────────────────────────────────────── */}
        <div className="flex flex-col gap-4 self-start">
          {o.ship_to && (
            <section className="rounded-2xl bg-card p-6">
              <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
                {t('order:deliveryTo')}
              </h2>
              <address className="mt-3 text-sm not-italic leading-relaxed text-ink-body">
                {o.ship_to.first_name} {o.ship_to.last_name}
                <br />
                {o.ship_to.street}
                <br />
                {o.ship_to.city} {o.ship_to.postal_code}, {o.ship_to.country}
                <br />
                {o.ship_to.phone}
              </address>
              {o.leave_with_neighbour && (
                <p className="mt-2 text-xs text-ink-soft">{t('checkout:address.neighbour')}</p>
              )}
              {o.delivery_note && (
                <p className="mt-2 text-xs text-ink-soft">“{o.delivery_note}”</p>
              )}
            </section>
          )}

          <section className="rounded-2xl bg-card p-6">
            <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
              {t('order:payment')}
            </h2>
            <p className="mt-3 text-sm text-ink-body">
              {t(`order:method.${o.payment_method}`)}
            </p>
            <p className="mt-1 text-xs text-ink-soft">
              {t(`order:paymentStatus.${o.payment_status}`)}
            </p>
          </section>

          {/* Reorder lives HERE, not on the history rows (Aug 2026):
              repeating a basket is a decision about its contents, and this
              page is the one place that lists every line. Settled orders
              only — a delivered order is a known-good basket, a cancelled
              one may be; live orders' jars are already on the way. */}
          {(o.status === 'delivered' || o.status === 'cancelled') && (
            <ReorderCard orderId={o.id} />
          )}

          {/* F2: self-service cancel, pending only — past that window the
              arrow belongs to the shop, and the 409 branch below says so.
              The canvas draws no cancel control anywhere, so the design is
              ours: the address book's armed-button pattern, one control
              that asks again and disarms after 3 s. */}
          {o.status === 'pending' && <CancelOrder orderId={o.id} />}

          <Link
            to={localePath('/account/orders')}
            className="text-sm font-semibold text-brand-ink hover:underline"
          >
            {t('order:allOrders')}
          </Link>
        </div>
      </div>
    </Shell>
  )
}

/** The whole basket back into the cart, with the merge report rendered
 *  right under the button — additions and every skip, reason named. */
function ReorderCard({ orderId }: { orderId: number }) {
  const { t } = useTranslation()
  const reorder = useReorder()

  return (
    <section className="rounded-2xl bg-card p-6">
      <Button
        variant="outline"
        fullWidth
        disabled={reorder.isPending}
        onClick={() => reorder.mutate(orderId)}
      >
        {t('account:ordersScreen.reorder')}
      </Button>
      {reorder.data && <ReorderReport report={reorder.data} />}
      {reorder.isError && (
        <p role="alert" className="mt-2 text-xs text-danger">
          {t('common:state.loadFailed')}
        </p>
      )}
    </section>
  )
}

function CancelOrder({ orderId }: { orderId: number }) {
  const { t } = useTranslation()
  const cancel = useCancelOrder()

  const [arming, setArming] = useState(false)
  const disarm = useRef<ReturnType<typeof setTimeout>>(undefined)
  useEffect(() => () => clearTimeout(disarm.current), [])

  const click = () => {
    if (arming) {
      clearTimeout(disarm.current)
      setArming(false)
      cancel.mutate(orderId)
      return
    }
    setArming(true)
    disarm.current = setTimeout(() => setArming(false), 3000)
  }

  // The 409 means the hive confirmed while this page was open; the hook
  // already refetched the order, so the tracker above tells the new truth
  // and this line explains why the button stopped working.
  const tooLate =
    cancel.error instanceof ApiError && cancel.error.code === 'too_late_to_cancel'

  return (
    <section className="rounded-2xl bg-card p-6">
      <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
        {t('order:cancel.title')}
      </h2>
      <p className="mt-2 text-xs leading-relaxed text-ink-soft">
        {t('order:cancel.blurb')}
      </p>
      <button
        type="button"
        onClick={click}
        disabled={cancel.isPending}
        className={cx(
          'mt-3 text-sm hover:underline disabled:opacity-50',
          arming ? 'font-bold text-danger' : 'font-semibold text-ink-faint hover:text-danger',
        )}
      >
        {arming ? t('order:cancel.confirm') : t('order:cancel.button')}
      </button>
      {tooLate && (
        <p role="alert" className="mt-2 text-xs text-danger">
          {t('order:cancel.tooLate')}
        </p>
      )}
      {cancel.isError && !tooLate && (
        <p role="alert" className="mt-2 text-xs text-danger">
          {t('common:state.loadFailed')}
        </p>
      )}
    </section>
  )
}

function Row({ label, value, muted }: { label: string; value: string; muted?: boolean }) {
  return (
    <div className={muted ? 'flex justify-between text-ink-muted' : 'flex justify-between'}>
      <dt className={muted ? undefined : 'text-ink-soft'}>{label}</dt>
      <dd className={muted ? undefined : 'font-medium text-ink'}>{value}</dd>
    </div>
  )
}

// A1: this page now renders inside AccountLayout's pane, which owns the
// page padding — the old centred `mx-auto px-6 py-10` here would pad twice.
function Shell({ children }: { children: React.ReactNode }) {
  return <div className="max-w-5xl">{children}</div>
}
