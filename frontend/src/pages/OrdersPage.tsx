import { useState } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useMyOrders, useReorder } from '../api/hooks'
import type { Order, OrderStatus, ReorderResult } from '../api/types'
import { OrderTracker } from '../components/account/OrderTracker'
import { cx } from '../lib/cx'
import { formatMoney } from '../lib/format'
import { useLocale } from '../i18n/useLocale'

/**
 * /account/orders — canvas 07. The screen splits the history in two: the
 * newest still-moving order gets the big bordered card with the tracker,
 * everything else is a compact row. The three filter pills work on the
 * already-fetched list — seven orders is a filter() away, not an endpoint.
 */

// "On the way" is the canvas's word for the machine's three live states.
const ACTIVE: OrderStatus[] = ['pending', 'confirmed', 'shipped']
const isActive = (o: Order) => ACTIVE.includes(o.status)

type Filter = 'all' | 'active' | 'delivered'

// How many history rows show before "Show N older orders" (the canvas
// draws three).
const VISIBLE_ROWS = 3

export function OrdersPage() {
  const { t } = useTranslation()
  const orders = useMyOrders()
  const reorder = useReorder()
  const [filter, setFilter] = useState<Filter>('all')
  const [expanded, setExpanded] = useState(false)

  if (orders.isPending) {
    return <p className="text-ink-body">{t('common:state.loading')}</p>
  }
  if (orders.isError) {
    return <p className="text-danger">{t('common:state.loadFailed')}</p>
  }

  const all = orders.data
  const activeCount = all.filter(isActive).length
  const deliveredCount = all.filter((o) => o.status === 'delivered').length

  // The list arrives newest-first, so the first live order is the newest.
  const featured = all.find(isActive)

  const passesFilter = (o: Order) =>
    filter === 'all' || (filter === 'active' ? isActive(o) : o.status === 'delivered')

  const showFeatured = featured !== undefined && passesFilter(featured)
  const rows = all.filter((o) => passesFilter(o) && o !== featured)
  const visibleRows = expanded ? rows : rows.slice(0, VISIBLE_ROWS)
  const hiddenCount = rows.length - VISIBLE_ROWS

  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex flex-col gap-1.5">
          <h1 className="font-display text-display-md font-extrabold text-ink">
            {t('account:nav.orders')}
          </h1>
          <p className="text-[0.9375rem] text-ink-soft">{t('account:ordersScreen.subtitle')}</p>
        </div>
        {all.length > 0 && (
          <div className="flex flex-wrap gap-2">
            <FilterPill
              label={`${t('account:ordersScreen.filterAll')} · ${all.length}`}
              selected={filter === 'all'}
              onClick={() => setFilter('all')}
            />
            {/* A pill that would filter to nothing does not render — an
                empty state behind a button is a dead end we can prevent. */}
            {activeCount > 0 && (
              <FilterPill
                label={`${t('account:ordersScreen.filterActive')} · ${activeCount}`}
                selected={filter === 'active'}
                onClick={() => setFilter('active')}
              />
            )}
            {deliveredCount > 0 && (
              <FilterPill
                label={`${t('account:ordersScreen.filterDelivered')} · ${deliveredCount}`}
                selected={filter === 'delivered'}
                onClick={() => setFilter('delivered')}
              />
            )}
          </div>
        )}
      </div>

      {reorder.data && <ReorderReport report={reorder.data} />}

      <div className="mt-6 flex flex-col gap-5">
        {all.length === 0 && <p className="text-ink-body">{t('account:noOrders')}</p>}

        {showFeatured && featured && (
          <FeaturedOrder order={featured} />
        )}

        {visibleRows.length > 0 && (
          <div className="rounded-3xl bg-card px-6 py-2 sm:px-7">
            <ul className="divide-y divide-line-soft">
              {visibleRows.map((o) => (
                <HistoryRow
                  key={o.id}
                  order={o}
                  onReorder={() => reorder.mutate(o.id)}
                  reordering={reorder.isPending}
                />
              ))}
            </ul>
            {(hiddenCount > 0 || expanded) && (
              <button
                type="button"
                onClick={() => setExpanded((v) => !v)}
                className="w-full py-4 text-center font-display text-sm font-semibold text-brand-ink hover:underline"
              >
                {expanded
                  ? t('account:ordersScreen.showFewer')
                  : t('account:ordersScreen.showOlder', { count: hiddenCount })}
              </button>
            )}
          </div>
        )}
      </div>
    </>
  )
}

function FilterPill({
  label,
  selected,
  onClick,
}: {
  label: string
  selected: boolean
  onClick: () => void
}) {
  return (
    // aria-pressed, not aria-selected: these are toggle buttons filtering a
    // list in place, not tabs switching panels — there is no tabpanel here.
    <button
      type="button"
      aria-pressed={selected}
      onClick={onClick}
      className={cx(
        'rounded-full px-4 py-2 font-display text-[0.8125rem] font-semibold transition',
        selected
          ? 'bg-bark text-ink-on-dark'
          : 'border-[1.5px] border-line text-ink-strong hover:border-line-strong',
      )}
    >
      {label}
    </button>
  )
}

/** The newest still-moving order: the canvas's orange-bordered card with
 *  the tracker and the items band. */
function FeaturedOrder({ order }: { order: Order }) {
  const { t } = useTranslation()
  const { locale, localePath } = useLocale()

  const itemCount = order.items.reduce((sum, it) => sum + it.qty, 0)
  const placed = new Date(order.created_at).toLocaleDateString(locale, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })

  return (
    <article className="flex flex-col gap-6 rounded-3xl border-2 border-brand bg-card p-6 sm:p-7">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="flex flex-col gap-1">
          <div className="flex flex-wrap items-center gap-3">
            <h2 className="font-display text-lg font-bold text-ink">
              {t('account:orderNumber', { id: order.id })}
            </h2>
            <StatusChip status={order.status} />
          </div>
          <p className="text-sm text-ink-soft">
            {t('account:ordersScreen.placed', { date: placed })} ·{' '}
            {t('account:ordersScreen.itemCount', { count: itemCount })}
            {order.has_cold_chain && <> · {t('account:ordersScreen.chilled')}</>}
          </p>
        </div>
        {/* The order's ONE currency (§1.3) — no second line here on purpose. */}
        <p className="font-display text-xl font-extrabold text-ink">
          {formatMoney(order.total_minor, order.currency)}
        </p>
      </div>

      <OrderTracker order={order} />

      <div className="flex flex-wrap items-center justify-between gap-4 rounded-2xl bg-panel px-5 py-4">
        <p className="min-w-0 flex-1 truncate text-sm text-ink-strong">
          {order.items.map((it) => it.name).join(' · ')}
        </p>
        {/* The canvas's "Track parcel" has nothing to track (§1.3); the
            card's one action is the door to the order's own page. */}
        <Link
          to={localePath(`/account/orders/${order.id}`)}
          className="rounded-full bg-brand-ink px-5 py-2.5 font-display text-sm font-semibold text-ink-on-dark transition hover:opacity-90"
        >
          {t('account:ordersScreen.details')}
        </Link>
      </div>
    </article>
  )
}

/** One compact history row: id + date, summary, total, status, reorder. */
function HistoryRow({
  order,
  onReorder,
  reordering,
}: {
  order: Order
  onReorder: () => void
  reordering: boolean
}) {
  const { t } = useTranslation()
  const { locale, localePath } = useLocale()

  const date = new Date(order.created_at).toLocaleDateString(locale, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })

  return (
    <li className="flex flex-col gap-3 py-5 sm:grid sm:grid-cols-[1.2fr_1fr_auto] sm:items-center sm:gap-5">
      <div className="flex flex-col gap-0.5">
        <Link
          to={localePath(`/account/orders/${order.id}`)}
          className="font-display text-[0.9375rem] font-bold text-ink hover:text-brand-ink hover:underline"
        >
          {t('account:orderNumber', { id: order.id })}
        </Link>
        <span className="text-[0.8125rem] text-ink-soft">
          {date} · {t('account:ordersScreen.itemCount', {
            count: order.items.reduce((sum, it) => sum + it.qty, 0),
          })}
        </span>
      </div>
      <p className="truncate text-sm text-ink-soft">
        {order.items.map((it) => it.name).join(' · ')}
      </p>
      <div className="flex items-center gap-3 sm:justify-end">
        <span className="font-display text-[0.9375rem] font-extrabold text-ink">
          {formatMoney(order.total_minor, order.currency)}
        </span>
        <StatusChip status={order.status} />
        {/* Reorder only where it makes sense: a delivered order is a known
            good basket; a cancelled one may be too. Live orders don't offer
            it — their jars are already on the way. */}
        {(order.status === 'delivered' || order.status === 'cancelled') && (
          <button
            type="button"
            onClick={onReorder}
            disabled={reordering}
            className="rounded-full border-[1.5px] border-line px-4 py-2 font-display text-[0.8125rem] font-semibold text-ink transition hover:border-line-strong disabled:opacity-50"
          >
            {t('account:ordersScreen.reorder')}
          </button>
        )}
      </div>
    </li>
  )
}

function StatusChip({ status }: { status: OrderStatus }) {
  const { t } = useTranslation()
  return (
    <span
      className={cx(
        'rounded-full px-3 py-1 font-display text-xs font-bold',
        // The canvas's chip colors: honey for a moving order, the soft
        // green for delivered; quiet for the rest. The green pair is the
        // canvas's own (#4C7A3D on #EAF2E3, 4.7:1 — passes AA).
        status === 'delivered' && 'bg-[#EAF2E3] text-[#4C7A3D]',
        (status === 'shipped' || status === 'confirmed') && 'bg-honey text-ink',
        status === 'pending' && 'bg-panel text-ink-strong',
        status === 'cancelled' && 'bg-panel text-ink-muted',
      )}
    >
      {t(`account:status.${status}`)}
    </span>
  )
}

/** What the reorder merge did — announced politely, with the skips named. */
function ReorderReport({ report }: { report: ReorderResult }) {
  const { t } = useTranslation()
  const { localePath } = useLocale()

  const added = report.lines.reduce((sum, l) => sum + l.qty, 0)
  const issues = report.lines.filter((l) => l.issue)

  return (
    <div role="status" className="mt-5 flex flex-col gap-1.5 rounded-2xl bg-honey/25 px-5 py-4">
      <p className="text-sm font-semibold text-ink">
        {added > 0
          ? t('account:ordersScreen.reorderAdded', { count: added })
          : t('account:ordersScreen.reorderNothing')}{' '}
        {added > 0 && (
          <Link to={localePath('/cart')} className="font-semibold text-brand-ink hover:underline">
            {t('account:ordersScreen.viewCart')}
          </Link>
        )}
      </p>
      {issues.map((l) => (
        <p key={l.name + l.label} className="text-[0.8125rem] text-ink-body">
          {l.name} ({l.label}) — {t(`account:ordersScreen.issue.${l.issue}`)}
        </p>
      ))}
    </div>
  )
}
