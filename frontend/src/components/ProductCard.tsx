import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { Product } from '../api/types'
import { useAddToCartFlash } from '../lib/useAddToCartFlash'
import { useHoverSlideshow } from '../lib/useHoverSlideshow'
import { useLocale } from '../i18n/useLocale'
import { Price } from './ui/Price'
import { cx } from '../lib/cx'
import { Badge, Stars } from './ui'
import { WishlistHeart } from './WishlistHeart'

/**
 * The design's product card: image slot with a badge and a wishlist heart,
 * a category eyebrow, the name, a "size · benefit" line, the price and an
 * Add button.
 *
 * Two departures from the mock, both recorded here rather than in a comment
 * nobody reads later:
 *
 *  1. The "size · benefit" line names a benefit from the TAXONOMY (Energy,
 *     Immunity, Skin…) where the mock writes a per-product phrase ("Natural
 *     energy", "Balms & candles"). Those would be two vocabularies for one
 *     slot; E2 models the taxonomy, because that is what the sidebar has to
 *     filter on. Slightly less evocative, one source of truth.
 *  2. The price is the CHEAPEST variant, labelled "from" when there is more
 *     than one. The mock draws a bare price because it shows one size per
 *     product; royal jelly has three, and an unlabelled $32 next to a
 *     product that also sells for $105 would be a lie of omission.
 *
 * The whole card is not one big <Link>. The mock's card holds two separate
 * controls (Add, the heart) besides the product link, and nesting
 * interactive elements inside an anchor is invalid HTML that browsers and
 * screen readers resolve differently. Only the NAME is the link, stretched
 * over the card with an ::after overlay so the whole surface is still
 * clickable — and the two buttons sit above it in the stacking order.
 */
export function ProductCard({
  product,
  onAdd,
  layout = 'compact',
}: {
  product: Product
  /**
   * Absent for a signed-out visitor; the button is disabled then. When the
   * promise resolves to a count, the button flashes "In cart: N" — the
   * confirmation state the mock never draws (§6 exception 2).
   */
  onAdd?: (product: Product) => void | Promise<number>
  /** 'compact' is the shop grid, 'feature' the home page's roomier card. */
  layout?: 'compact' | 'feature'
}) {
  const { t } = useTranslation()
  const { localePath } = useLocale()

  // The transient "In cart: N" flash — shared with the wishlist's card via
  // one hook, so the confirmation behaviour cannot drift between grids.
  const { addedQty, handleAdd } = useAddToCartFlash(onAdd)

  // Hovering the card cycles its photos (decision #99) — same shared-hook
  // arrangement as the flash, and inert on touch screens, under
  // prefers-reduced-motion, and for single-photo products.
  const images = product.images
  const slideshow = useHoverSlideshow(images.length)

  // Variants arrive sorted by price, so the first is the "from" price.
  const cheapest = product.variants[0]
  const hasChoice = product.variants.length > 1
  const inStock = product.variants.some((v) => v.stock_qty > 0)
  const benefit = product.benefits[0]

  return (
    <article
      // The hover handlers live on the CARD, not the image slot: the
      // stretched-link overlay (the ::after below) sits above the image, so
      // pointer events over the photo actually target the anchor — a
      // listener on the image div would never hear them. On the article,
      // every descendant counts, overlay included. Found by the e2e test,
      // which hovers with a real pointer.
      onMouseEnter={slideshow.onMouseEnter}
      onMouseLeave={slideshow.onMouseLeave}
      className={cx(
        'group relative flex h-full flex-col gap-3 rounded-xl bg-card',
        layout === 'feature' ? 'gap-3.5 p-5' : 'p-4.5',
      )}
    >
      <div
        className={cx(
          'relative flex items-center justify-center overflow-hidden rounded-lg bg-panel',
          layout === 'feature' ? 'h-52' : 'h-50',
        )}
      >
        {images[0] ? (
          slideshow.warm ? (
            // After the first hover every photo is mounted, stacked, one
            // visible — the browser fetches the hidden ones right then, so
            // the cycle never shows a blank frame. Before it, the plain
            // hero below keeps a twelve-card grid at twelve requests.
            images.map((img, i) => (
              <img
                key={img.url}
                src={img.url}
                alt=""
                className={cx(
                  'absolute inset-0 size-full object-cover',
                  i !== slideshow.index && 'invisible',
                )}
              />
            ))
          ) : (
            <img
              src={images[0].url}
              alt=""
              className="size-full object-cover"
              loading="lazy"
            />
          )
        ) : (
          // The mock draws a hatched placeholder with the shot description in
          // it. Real products have no photos yet, so the slot keeps the
          // card's geometry instead of collapsing. alt="" / aria-hidden: the
          // product name is right underneath, so announcing this too would
          // just repeat it.
          <span
            aria-hidden
            className="px-4 text-center font-mono text-2xs uppercase tracking-label text-ink-muted"
          >
            {product.slug.replace(/-/g, ' ')}
          </span>
        )}

        {product.badge && (
          <Badge tone={product.badge_tone} className="absolute left-3 top-3">
            {/* A KEY from the API, looked up here — the backend never sends
                English prose, so the same response renders in all three
                languages (backend/migrations/000009). */}
            {t(`catalog:badge.${product.badge}`)}
          </Badge>
        )}

        {/* E8: the heart is live — one shared component owns the state, so
            this card and the product page can never disagree about it. */}
        <WishlistHeart productId={product.id} className="absolute right-3 top-3 z-10" />

        {/* Position dots, only while the slideshow runs. aria-hidden: purely
            decorative — the photos are alt="" (the name sits right below),
            so narrating "photo 2 of 3" would describe nothing. Nudged up
            when the sold-out chip occupies the bottom edge. */}
        {slideshow.cycling && (
          <div
            aria-hidden
            className={cx(
              'absolute inset-x-0 z-10 flex justify-center gap-1.5',
              inStock ? 'bottom-2.5' : 'bottom-13',
            )}
          >
            {images.map((img, i) => (
              <span
                key={img.url}
                className={cx(
                  'size-1.5 rounded-full transition',
                  i === slideshow.index ? 'bg-bark' : 'bg-card/80',
                )}
              />
            ))}
          </div>
        )}

        {!inStock && (
          <span className="absolute inset-x-3 bottom-3 rounded-full bg-bark/90 px-3 py-1.5 text-center text-2xs font-semibold uppercase tracking-label text-ink-on-dark">
            {t('catalog:outOfStock')}
          </span>
        )}
      </div>

      {/* The mock's eyebrow is the CATEGORY, and it arrives already resolved
          into the reader's language — the card would otherwise have to fetch
          /categories and map category_id by hand. */}
      <span className="font-display text-2xs font-semibold uppercase tracking-eyebrow text-ink-muted">
        {product.category_name}
      </span>

      <h3
        className={cx(
          'font-display font-bold text-ink',
          layout === 'feature' ? 'text-lg' : 'text-base',
        )}
      >
        <Link
          to={localePath(`/products/${product.slug}`)}
          // The stretched-link trick: the ::after box covers the card, so
          // the whole surface is clickable while only this anchor is the
          // link. `relative` + `z-10` on the buttons keeps them on top.
          className="after:absolute after:inset-0 after:content-[''] hover:text-brand-ink"
        >
          {product.name}
        </Link>
      </h3>

      {/* Rendered only once a product HAS ratings: an empty star row on
          every card in a new shop is five grey shapes saying nothing, and it
          pushes the price line down for no information. */}
      {product.rating_count > 0 && (
        <Stars rating={product.rating_avg} count={product.rating_count} />
      )}

      {layout === 'feature' ? (
        <p className="text-sm leading-relaxed text-ink-soft">{product.description}</p>
      ) : (
        cheapest && (
          <p className="text-xs text-ink-soft">
            {benefit
              ? t('catalog:sizeAndBenefit', {
                  size: cheapest.label,
                  benefit: benefit.name,
                })
              : cheapest.label}
          </p>
        )
      )}

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
          onClick={() => void handleAdd(product)}
          className={cx(
            'relative z-10 shrink-0 rounded-full font-display text-xs font-semibold transition',
            'disabled:pointer-events-none disabled:opacity-50',
            // While the flash shows, the compact card's outline button goes
            // solid too — the count reads as a confirmation, not a relabel.
            addedQty !== null || layout === 'feature'
              ? 'bg-bark px-4.5 py-2.5 text-ink-on-dark hover:opacity-90'
              : 'border-[1.5px] border-bark px-4 py-2.5 text-ink hover:bg-bark hover:text-ink-on-dark',
          )}
        >
          {/* The mock writes a bare "Add" on the compact card; one action,
              one name everywhere was preferred over the shorter chip.
              Keying the span by count remounts it on every successful
              click, so the pop replays even when only the number changed. */}
          <span
            key={addedQty ?? 'resting'}
            className={cx(
              'inline-block',
              addedQty !== null && 'animate-pop motion-reduce:animate-none',
            )}
          >
            {addedQty !== null
              ? t('catalog:inCart', { count: addedQty })
              : t('catalog:addToCart')}
          </span>
        </button>

        {/* A button changing its own text is not announced by screen
            readers — the live region is what says "In cart: 2" out loud. */}
        <span aria-live="polite" className="sr-only">
          {addedQty !== null ? t('catalog:inCart', { count: addedQty }) : ''}
        </span>
      </div>
    </article>
  )
}
