import { useTranslation } from 'react-i18next'
import type { Cart, Preview } from '../../api/types'
import { useSetCartItem } from '../../api/hooks'
import { formatMoney } from '../../lib/format'

/**
 * The honey banner from the design's cart screen: "$8 away from free
 * shipping" with an upsell button. The mock draws only that one state; the
 * other two are ours to design (§6 exception 2):
 *
 *  - counting: the gap, the progress BAR (the plan asks for one even though
 *    the mock omits it), and the server's suggestion of one product that
 *    closes the gap in a click.
 *  - unlocked: the threshold (or a free-shipping code) waived the base.
 *  - first order: the hive-club welcome — free base delivery, no counting.
 *
 * Every number here is the preview's. The client's only arithmetic is the
 * bar's width, which is display math — the money math lives in Go.
 */
export function FreeShippingBanner({ preview, cart }: { preview: Preview; cart: Cart }) {
  const { t } = useTranslation()
  const setItem = useSetCartItem()

  const remaining = preview.free_shipping_remaining_minor
  const threshold = preview.free_shipping_threshold_minor

  // Neither counting nor waived: this market has no threshold — no promise
  // to draw a banner about.
  if (remaining === undefined && !preview.base_shipping_waived) return null

  if (remaining === undefined || threshold === undefined) {
    // Waived. Say why — a kept promise is worth naming.
    const [title, blurb] = preview.first_delivery_free
      ? [t('cart:progress.firstOrder'), t('cart:progress.firstOrderBlurb')]
      : [t('cart:progress.unlocked'), t('cart:progress.unlockedBlurb')]
    return (
      <div className="flex flex-col gap-1 rounded-2xl bg-honey px-7 py-6">
        <p className="font-display text-lg font-bold text-ink">{title}</p>
        <p className="text-sm text-ink-strong">{blurb}</p>
      </div>
    )
  }

  const percent = Math.round(((threshold - remaining) / threshold) * 100)
  // If the suggestion is somehow already a cart line, "add one" means one
  // MORE — PUT sets an absolute quantity, so the button computes it.
  const upsellQty = preview.upsell
    ? (cart.items.find((it) => it.variant_id === preview.upsell!.variant_id)?.qty ?? 0) + 1
    : 1

  return (
    <div className="flex flex-wrap items-center justify-between gap-6 rounded-2xl bg-honey px-7 py-6">
      <div className="flex min-w-52 flex-1 flex-col gap-2">
        <p className="font-display text-lg font-bold text-ink">
          {t('cart:progress.away', {
            amount: formatMoney(remaining, preview.currency),
          })}
        </p>
        <p className="text-sm text-ink-strong">{t('cart:progress.blurb')}</p>
        <div
          role="progressbar"
          aria-label={t('cart:progress.label')}
          aria-valuenow={percent}
          aria-valuemin={0}
          aria-valuemax={100}
          className="mt-1 h-2 w-full max-w-80 overflow-hidden rounded-full bg-ink/15"
        >
          <div className="h-full rounded-full bg-bark" style={{ width: `${percent}%` }} />
        </div>
      </div>

      {preview.upsell && (
        <button
          type="button"
          disabled={setItem.isPending}
          onClick={() =>
            setItem.mutate({ variantId: preview.upsell!.variant_id, qty: upsellQty })
          }
          className="whitespace-nowrap rounded-full bg-bark px-5 py-3.5 font-display text-sm font-semibold text-ink-on-dark transition hover:bg-bark-soft disabled:opacity-50"
        >
          {t('cart:progress.add', {
            name: preview.upsell.name,
            price: formatMoney(preview.upsell.price_minor, preview.currency),
          })}
        </button>
      )}
    </div>
  )
}
