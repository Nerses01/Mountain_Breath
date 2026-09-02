import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useAddWishlistToCart, useQuickAdd, useWishlist } from '../api/hooks'
import { ReorderReport } from '../components/account/ReorderReport'
import { WishlistCard } from '../components/account/WishlistCard'
import { HeartIcon } from '../components/ui/icons'
import { cx } from '../lib/cx'
import { formatMoney } from '../lib/format'
import { useAddToCartFlash } from '../lib/useAddToCartFlash'
import { useCurrency } from '../lib/useCurrency'
import { useLocale } from '../i18n/useLocale'

/**
 * /account/wishlist — canvas 08: the saved shelf with its own card, a
 * header that totals it, one-tap add-all, and the dashed "save more" slot.
 *
 * The worth-total is DISPLAY math, not money math: the sum of the same
 * per-card prices the grid renders, formatted once. Nothing is charged
 * from it — the cart's server-side preview stays the only calculator.
 */
export function WishlistPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { currency } = useCurrency()
  const wishlist = useWishlist(true)
  const quickAdd = useQuickAdd()
  const addAll = useAddWishlistToCart()
  // The button's own confirmation — the count comes from the server's merge
  // REPORT, not from how many cards are on screen (sold-out lines add 0).
  const { addedQty, flash } = useAddToCartFlash()

  if (wishlist.isPending) {
    return <p className="text-ink-body">{t('common:state.loading')}</p>
  }
  if (wishlist.isError) {
    return <p className="text-danger">{t('common:state.loadFailed')}</p>
  }

  const entries = wishlist.data ?? []
  // The card's own "from" price per entry (variants arrive price-sorted),
  // summed in the market the page is already showing.
  const worth = entries.reduce((sum, e) => sum + (e.variants[0]?.price_minor ?? 0), 0)
  const anyInStock = entries.some((e) => e.variants.some((v) => v.stock_qty > 0))

  return (
    <>
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex flex-col gap-1.5">
          <h1 className="font-display text-display-md font-extrabold text-ink">
            {t('account:nav.wishlist')}
          </h1>
          {entries.length > 0 && (
            <p className="text-[0.9375rem] text-ink-soft">
              {t('account:wishlist.summary', {
                count: entries.length,
                total: formatMoney(worth, currency),
              })}
            </p>
          )}
        </div>
        {entries.length > 0 && (
          <button
            type="button"
            onClick={() =>
              addAll.mutate(undefined, {
                onSuccess: (report) => {
                  const added = report.lines.reduce((sum, l) => sum + l.qty, 0)
                  if (added > 0) flash(added)
                },
              })
            }
            disabled={!anyInStock || addAll.isPending}
            className="rounded-full bg-bark px-5.5 py-3 font-display text-sm font-semibold text-ink-on-dark transition hover:opacity-90 disabled:opacity-50"
          >
            {/* The flash says the TOTAL the server added; the report below
                stays the detailed, per-line feedback. */}
            <span
              key={addedQty ?? 'resting'}
              className={cx(
                'inline-block',
                addedQty !== null && 'animate-pop motion-reduce:animate-none',
              )}
            >
              {addedQty !== null
                ? t('account:ordersScreen.reorderAdded', { count: addedQty })
                : t('account:wishlist.addAll')}
            </span>
          </button>
        )}
      </div>

      {addAll.data && <ReorderReport report={addAll.data} />}

      {entries.length === 0 ? (
        <p className="mt-4 text-ink-body">
          {t('account:wishlist.empty')}{' '}
          <Link to={localePath('/shop')} className="font-semibold text-brand-ink hover:underline">
            {t('cart:browse')}
          </Link>
        </p>
      ) : (
        <div className="mt-7 grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {entries.map((e) => (
            <WishlistCard key={e.id} entry={e} onAdd={quickAdd} />
          ))}

          {/* The canvas's dashed slot: the grid's last cell sells the way
              more cards get here. */}
          <div className="flex min-h-70 flex-col items-center justify-center gap-2.5 rounded-xl border-2 border-dashed border-line-strong p-6 text-center">
            <span
              aria-hidden
              className="flex size-11 items-center justify-center rounded-full bg-card text-ink-faint"
            >
              <HeartIcon size={19} />
            </span>
            <p className="font-display text-[0.9375rem] font-bold text-ink">
              {t('account:wishlist.saveMoreTitle')}
            </p>
            <p className="max-w-52 text-[0.8125rem] text-ink-muted">
              {t('account:wishlist.saveMoreBlurb')}
            </p>
            <Link
              to={localePath('/shop')}
              className="font-display text-sm font-semibold text-brand-ink hover:underline"
            >
              {t('account:wishlist.browseShelf')}
            </Link>
          </div>
        </div>
      )}
    </>
  )
}
