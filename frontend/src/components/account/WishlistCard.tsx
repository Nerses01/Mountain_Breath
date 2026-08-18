import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { Product, WishlistEntry } from '../../api/types'
import { cx } from '../../lib/cx'
import { useLocale } from '../../i18n/useLocale'
import { Badge } from '../ui'
import { Price } from '../ui/Price'
import { WishlistHeart } from '../WishlistHeart'

/**
 * A3 (canvas 08): the wishlist's OWN card — not the shop's ProductCard.
 * The shelf card answers "should I buy this?"; this one answers "you
 * already chose this, when, and can you have it now?" — so the eyebrow,
 * stars and benefit line go, and the saved-ago line arrives.
 *
 * What stays identical to ProductCard is deliberate too: the heart is the
 * same shared WishlistHeart (un-hearting IS removal — no second delete
 * control), the stretched-link trick makes the surface clickable while
 * only the name is the anchor, and the sold-out state disables Add rather
 * than drawing the deferred "Notify me" (decision #6: back-in-stock is
 * parked; a dead button would be the decorative control this project
 * refuses — recorded as a canvas departure in PLAN_ACCOUNT.md).
 */
export function WishlistCard({
  entry,
  onAdd,
}: {
  entry: WishlistEntry
  onAdd?: (product: Product) => void | Promise<number>
}) {
  const { t } = useTranslation()
  const { locale, localePath } = useLocale()

  const cheapest = entry.variants[0]
  const hasChoice = entry.variants.length > 1
  const inStock = entry.variants.some((v) => v.stock_qty > 0)

  return (
    <article className="group relative flex h-full flex-col gap-3 rounded-xl bg-card p-4.5">
      <div className="relative flex h-42 items-center justify-center overflow-hidden rounded-lg bg-panel">
        {entry.image_url ? (
          <img src={entry.image_url} alt="" className="size-full object-cover" loading="lazy" />
        ) : (
          <span
            aria-hidden
            className="px-4 text-center font-mono text-2xs uppercase tracking-label text-ink-muted"
          >
            {entry.slug.replace(/-/g, ' ')}
          </span>
        )}

        {entry.badge && (
          <Badge tone={entry.badge_tone} className="absolute left-3 top-3">
            {t(`catalog:badge.${entry.badge}`)}
          </Badge>
        )}
        <WishlistHeart productId={entry.id} className="absolute right-3 top-3 z-10" />
        {!inStock && (
          <span className="absolute inset-x-3 bottom-3 rounded-full bg-bark/90 px-3 py-1.5 text-center text-2xs font-semibold uppercase tracking-label text-ink-on-dark">
            {t('catalog:outOfStock')}
          </span>
        )}
      </div>

      <h3 className="font-display text-base font-bold text-ink">
        <Link
          to={localePath(`/products/${entry.slug}`)}
          className="after:absolute after:inset-0 after:content-[''] hover:text-brand-ink"
        >
          {entry.name}
        </Link>
      </h3>

      {/* size · saved N ago — the card's one metadata line. */}
      <p className="text-xs text-ink-soft">
        {cheapest?.label}
        {' · '}
        {t('account:wishlist.savedAgo', { ago: relativeTime(entry.saved_at, locale) })}
      </p>

      <div className="mt-auto flex items-end justify-between gap-3 pt-2">
        <Price
          prices={cheapest?.prices}
          primaryMinor={cheapest?.price_minor}
          size="lg"
          format={hasChoice ? (price) => t('catalog:priceFrom', { price }) : undefined}
        />
        <button
          type="button"
          disabled={!inStock || !onAdd}
          onClick={() => void onAdd?.(entry)}
          className={cx(
            'relative z-10 shrink-0 rounded-full bg-bark px-4.5 py-2.5 font-display text-xs font-semibold text-ink-on-dark transition hover:opacity-90',
            'disabled:pointer-events-none disabled:opacity-50',
          )}
        >
          {t('catalog:addToCart')}
        </button>
      </div>
    </article>
  )
}

/**
 * "2 weeks ago" in the reader's language. Intl.RelativeTimeFormat speaks
 * all three locales for free — the unit choice (day/week/month) is ours,
 * the words are the platform's. Under a day rounds to "today"-ish "0 days
 * ago" — fine for a wishlist, where precision is nostalgia, not billing.
 */
function relativeTime(iso: string, locale: string): string {
  const days = Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000)
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  if (days < 7) return rtf.format(-days, 'day')
  if (days < 30) return rtf.format(-Math.floor(days / 7), 'week')
  if (days < 365) return rtf.format(-Math.floor(days / 30), 'month')
  return rtf.format(-Math.floor(days / 365), 'year')
}
