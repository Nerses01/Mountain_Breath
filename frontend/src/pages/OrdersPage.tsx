import { useState } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useMyOrders } from '../api/hooks'
import type { Order, OrderStatus } from '../api/types'
import { OrderTracker } from '../components/account/OrderTracker'
import { Pagination } from '../components/ui'
import { cx } from '../lib/cx'
import { formatMoney } from '../lib/format'
import { useLocale } from '../i18n/useLocale'

/**
 * /account/orders — canvas 07. The screen splits the history in two: the
 * newest still-moving order gets the big bordered card with the tracker,
 * everything else is a compact row. The three filter pills and the pager
 * work on the already-fetched list — a dozen orders is a filter() and a
 * slice() away, not an endpoint.
 *
 * Rows carry no Reorder button (the canvas draws one, Aug 2026 decision):
 * repeating a basket is a decision about its CONTENTS, and the row shows a
 * truncated summary — the detail page, which shows every line, owns the
 * action.
 */

// "On the way" is the canvas's word for the machine's three live states.
const ACTIVE: OrderStatus[] = ['pending', 'confirmed', 'shipped']
const isActive = (o: Order) => ACTIVE.includes(o.status)

type Filter = 'all' | 'active' | 'delivered'

// History rows per page. The canvas draws three rows and a "Show 5 older
// orders" disclosure; numbered pages replaced it (Aug 2026) so a long
// history stays walkable instead of growing one ever-taller column.
const ROWS_PER_PAGE = 3

export function OrdersPage() {
  const { t } = useTranslation()
  const orders = useMyOrders()
  const [filter, setFilter] = useState<Filter>('all')
  const [page, setPage] = useState(1)

  // One mover for both states: switching filters re-slices the list, so the
  // page number must restart — page 3 of "all" means nothing under
  // "delivered".
  const pickFilter = (f: Filter) => {
    setFilter(f)
    setPage(1)
  }

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
  const pageCount = Math.max(1, Math.ceil(rows.length / ROWS_PER_PAGE))
  const visibleRows = rows.slice((page - 1) * ROWS_PER_PAGE, page * ROWS_PER_PAGE)

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
              onClick={() => pickFilter('all')}
            />
            {/* A pill that would filter to nothing does not render — an
                empty state behind a button is a dead end we can prevent. */}
            {activeCount > 0 && (
              <FilterPill
                label={`${t('account:ordersScreen.filterActive')} · ${activeCount}`}
                selected={filter === 'active'}
                onClick={() => pickFilter('active')}
              />
            )}
            {deliveredCount > 0 && (
              <FilterPill
                label={`${t('account:ordersScreen.filterDelivered')} · ${deliveredCount}`}
                selected={filter === 'delivered'}
                onClick={() => pickFilter('delivered')}
              />
            )}
          </div>
        )}
      </div>

      <div className="mt-6 flex flex-col gap-5">
        {all.length === 0 && <p className="text-ink-body">{t('account:noOrders')}</p>}

        {showFeatured && featured && (
          <FeaturedOrder order={featured} />
        )}

        {visibleRows.length > 0 && (
          <div className="rounded-3xl bg-card px-6 py-2 sm:px-7">
            <ul className="divide-y divide-line-soft">
              {visibleRows.map((o) => (
                <HistoryRow key={o.id} order={o} />
              ))}
            </ul>
            {pageCount > 1 && (
              <div className="border-t border-line-soft py-4">
                <Pagination page={page} pageCount={pageCount} onSelect={setPage} />
              </div>
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

/** One compact history row: id + date, summary, total, status. */
function HistoryRow({ order }: { order: Order }) {
  const { t } = useTranslation()
  const { locale, localePath } = useLocale()

  const date = new Date(order.created_at).toLocaleDateString(locale, {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
  })

  return (
    // A FIXED first column, not a fr share: every <li> is its own grid, and
    // fr units hand the id block ~45% of the row, floating the summary
    // toward the middle. A fixed track starts the summary at the same x in
    // every row — the tabular look the eye expects from a list of records.
    <li className="flex flex-col gap-3 py-5 sm:grid sm:grid-cols-[11.5rem_1fr_auto] sm:items-center sm:gap-5">
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
      {/* Min-widths + right/center alignment make the money and status read
          as COLUMNS across rows, while still growing for longer figures and
          the other languages' chip labels. tabular-nums: digits share one
          width, so totals align digit-for-digit. */}
      <div className="flex items-center gap-3 sm:justify-end">
        <span className="min-w-22 text-right font-display text-[0.9375rem] font-extrabold tabular-nums text-ink">
          {formatMoney(order.total_minor, order.currency)}
        </span>
        <StatusChip status={order.status} className="min-w-23 text-center" />
      </div>
    </li>
  )
}

function StatusChip({ status, className }: { status: OrderStatus; className?: string }) {
  const { t } = useTranslation()
  return (
    <span
      className={cx(
        className,
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

