import { useState } from 'react'
import { Link, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { ApiError } from '../api/client'
import {
  useCart,
  useMe,
  useProduct,
  useRelatedProducts,
  useSetCartItem,
} from '../api/hooks'
import type { ProductDetail, ProductVariant } from '../api/types'
import { ProductCard } from '../components/ProductCard'
import { Gallery } from '../components/product/Gallery'
import { Tabs, type Tab } from '../components/product/Tabs'
import {
  Badge,
  Breadcrumbs,
  Button,
  Card,
  CheckIcon,
  HeartIcon,
  IconButton,
  QtyStepper,
  SectionHeading,
} from '../components/ui'
import { useLocale } from '../i18n/useLocale'
import { cx } from '../lib/cx'
import { formatMoney } from '../lib/format'

/**
 * The design's product screen, rendered entirely from API data.
 *
 * Two departures from the mock, both recorded rather than silently applied:
 *
 *  1. NO REVIEWS TAB. The design draws "Reviews (64)" beside the other two
 *     and shows ★★★★★ under the category eyebrow. Reviews are E4 — shipping
 *     a tab that opens onto nothing, or a rating with no reviews behind it,
 *     would be worse than the design's shape arriving one phase late. This
 *     is the same call E1.5 made for the nav links to pages that did not
 *     exist yet.
 *  2. ONE PRICE, not two. E5 adds the AMD line the mock shows beneath.
 */
export function ProductPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { slug } = useParams<{ slug: string }>()

  const product = useProduct(slug ?? '')
  const related = useRelatedProducts(slug ?? '')

  const [variantId, setVariantId] = useState<number | null>(null)
  const [qty, setQty] = useState(1)

  const me = useMe()
  const cart = useCart(!!me.data)
  const addToCart = useSetCartItem()

  if (product.isPending) {
    return <PageShell>{t('common:state.loading')}</PageShell>
  }

  if (product.isError) {
    const notFound = product.error instanceof ApiError && product.error.status === 404
    return (
      <PageShell>
        <Card className="flex flex-col items-start gap-3">
          <p className="font-display text-lg font-bold text-ink">
            {notFound ? t('catalog:notFound') : t('common:state.loadFailed')}
          </p>
          <Link to={localePath('/shop')} className="text-brand-ink hover:underline">
            {t('catalog:back')}
          </Link>
        </Card>
      </PageShell>
    )
  }

  const p: ProductDetail = product.data
  const selected: ProductVariant | undefined =
    p.variants.find((v) => v.id === variantId) ?? p.variants[0]
  const inCartQty = cart.data?.items.find((it) => it.variant_id === selected?.id)?.qty ?? 0
  const soldOut = !selected || selected.stock_qty === 0

  const tabs: Tab[] = []
  if (p.usage_cards.length > 0) {
    tabs.push({
      id: 'how-to-take-it',
      label: t('product:tabs.howToTakeIt'),
      panel: (
        <div className="grid gap-6 md:grid-cols-3">
          {p.usage_cards.map((card) => (
            <Card key={card.kicker + card.title} className="flex flex-col gap-2 p-6.5">
              <span className="font-display text-sm font-extrabold text-brand-ink">
                {card.kicker}
              </span>
              <h3 className="font-display text-lg font-bold text-ink">{card.title}</h3>
              <p className="text-base leading-relaxed text-ink-body">{card.body}</p>
            </Card>
          ))}
        </div>
      ),
    })
  }
  if (p.storage_note) {
    tabs.push({
      id: 'storage',
      label: t('product:tabs.storage'),
      panel: (
        <Card className="p-6.5">
          <p className="max-w-200 text-base leading-relaxed text-ink-body">{p.storage_note}</p>
        </Card>
      ),
    })
  }

  return (
    <div className="mx-auto max-w-360 px-6 py-8 lg:px-14">
      <Breadcrumbs
        items={[
          { label: t('common:nav.home'), to: localePath('/') },
          { label: t('common:nav.shop'), to: localePath('/shop') },
          {
            label: p.category_name,
            to: `${localePath('/shop')}?category=${p.category_slug}`,
          },
          { label: p.name },
        ]}
      />

      <div className="mt-4 grid gap-12 lg:grid-cols-2">
        <div className="relative">
          <Gallery images={p.images} productName={p.name} />
          {p.badge && (
            <Badge tone={p.badge_tone} className="absolute left-4.5 top-4.5">
              {t(`catalog:badge.${p.badge}`)}
            </Badge>
          )}
        </div>

        <div className="flex flex-col gap-5">
          <span className="font-display text-xs font-semibold uppercase tracking-eyebrow text-ink-muted">
            {p.category_name}
          </span>

          <h1 className="font-display text-display-lg font-extrabold text-ink">{p.name}</h1>

          <p className="text-lg leading-relaxed text-ink-body">{p.description}</p>

          <p className="flex flex-wrap items-baseline gap-3.5">
            <span className="font-display text-display-md font-extrabold text-brand-ink">
              {formatMoney(selected?.price_minor ?? 0)}
            </span>
            {selected && (
              <span className="text-sm text-ink-soft">· {selected.label}</span>
            )}
          </p>

          {p.highlights.length > 0 && (
            <Card className="flex flex-col gap-3 p-6">
              <h2 className="font-display text-base font-bold uppercase tracking-label text-ink">
                {t('product:whatItDoes')}
              </h2>
              <ul className="flex flex-col gap-3">
                {p.highlights.map((h) => (
                  <li key={h.text} className="flex items-start gap-3">
                    <span
                      aria-hidden
                      className="mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-honey text-ink"
                    >
                      <CheckIcon className="size-3" />
                    </span>
                    <span className="text-base text-ink-strong">{h.text}</span>
                  </li>
                ))}
              </ul>
              {p.disclaimer && (
                <p className="pt-1 text-xs text-ink-muted">{p.disclaimer}</p>
              )}
            </Card>
          )}

          {p.variants.length > 0 && (
            <fieldset className="flex flex-col gap-2.5">
              {/* A fieldset + legend, not a heading and a row of buttons:
                  picking a size is choosing one of a set, and the legend is
                  what names that set for a screen reader. */}
              <legend className="mb-2.5 font-display text-sm font-bold text-ink">
                {t('product:size')}
              </legend>
              <div className="flex flex-wrap gap-2.5">
                {p.variants.map((v) => {
                  const isSelected = v.id === selected?.id
                  const out = v.stock_qty === 0
                  return (
                    <button
                      key={v.id}
                      type="button"
                      disabled={out}
                      aria-pressed={isSelected}
                      onClick={() => {
                        setVariantId(v.id)
                        setQty(1)
                      }}
                      className={cx(
                        'rounded-lg border-[1.5px] px-5 py-3 font-display text-sm font-semibold transition',
                        isSelected
                          ? 'border-2 border-brand bg-card text-ink'
                          : 'border-line text-ink-body hover:border-line-strong',
                        out && 'cursor-not-allowed line-through opacity-50',
                      )}
                    >
                      {/* The design's pill is "25 g · $32" — the size alone
                          makes a shopper compare by clicking. */}
                      {v.label} · {formatMoney(v.price_minor)}
                      {out && <span className="sr-only"> — {t('catalog:outOfStock')}</span>}
                    </button>
                  )
                })}
              </div>
            </fieldset>
          )}

          <div className="flex flex-wrap items-center gap-3.5">
            <QtyStepper
              value={qty}
              onChange={setQty}
              min={1}
              max={Math.max(selected?.stock_qty ?? 1, 1)}
              decreaseLabel={t('cart:decrease')}
              increaseLabel={t('cart:increase')}
            />

            {me.data ? (
              <Button
                size="lg"
                disabled={soldOut || addToCart.isPending}
                onClick={() =>
                  selected && addToCart.mutate({ variantId: selected.id, qty: inCartQty + qty })
                }
                className="flex-1"
              >
                {soldOut
                  ? t('catalog:outOfStock')
                  : addToCart.isPending
                    ? t('catalog:adding')
                    : // The price in the button label, as the mock draws it:
                      // it is the last thing read before committing.
                      t('product:addToCartWithPrice', {
                        price: formatMoney((selected?.price_minor ?? 0) * qty),
                      })}
              </Button>
            ) : (
              <Button size="lg" className="flex-1" disabled>
                {t('catalog:signInToBuy')}
              </Button>
            )}

            <IconButton label={t('common:actions.wishlist')} disabled className="size-13.5">
              <HeartIcon />
            </IconButton>
          </div>

          {inCartQty > 0 && (
            <Link
              to={localePath('/cart')}
              className="font-display text-sm font-semibold text-brand-ink hover:underline"
            >
              {t('catalog:inCart', { count: inCartQty })} →
            </Link>
          )}

          <dl className="grid gap-3 pt-1.5 sm:grid-cols-3">
            <MetaCard label={t('product:meta.harvest')} value={p.harvest_note} />
            <MetaCard label={t('product:meta.shipping')} value={p.shipping_note} />
            <MetaCard
              label={t('product:meta.lab')}
              value={p.lab_batch && t('product:meta.batch', { batch: p.lab_batch })}
            />
          </dl>
        </div>
      </div>

      {tabs.length > 0 && (
        <div className="mt-14">
          <Tabs tabs={tabs} label={t('product:tabs.label')} />
        </div>
      )}

      {(related.data?.length ?? 0) > 0 && (
        <section className="mt-14 flex flex-col gap-6 rounded-2xl bg-panel-soft p-11">
          <SectionHeading
            title={t('product:related')}
            size="sm"
            action={{ label: t('home:shelf.action'), to: localePath('/shop') }}
          />
          <div className="grid gap-5.5 sm:grid-cols-2 xl:grid-cols-4">
            {related.data?.map((r) => (
              <ProductCard key={r.id} product={r} />
            ))}
          </div>
        </section>
      )}
    </div>
  )
}

// A <dl> pair per card: these are term/definition, and saying so in markup
// costs nothing and tells a screen reader how to read the row.
function MetaCard({ label, value }: { label: string; value?: string | false }) {
  if (!value) return null
  return (
    <Card className="flex flex-col gap-1 p-4">
      <dt className="font-display text-sm font-bold text-ink">{label}</dt>
      <dd className="text-xs text-ink-soft">{value}</dd>
    </Card>
  )
}

function PageShell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-360 px-6 py-10 lg:px-14">{children}</div>
}
