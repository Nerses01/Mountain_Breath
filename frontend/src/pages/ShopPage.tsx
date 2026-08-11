import { useTranslation } from 'react-i18next'
import { useCatalogFacets, useMe, useProducts, useSetCartItem } from '../api/hooks'
import type { Product, ProductSort } from '../api/types'
import { ProductCard } from '../components/ProductCard'
import { Breadcrumbs, Button, Card } from '../components/ui'
import { PriceRange } from '../components/ui/PriceRange'
import { useLocale } from '../i18n/useLocale'
import { cx } from '../lib/cx'
import { PER_PAGE, useCatalogFilters } from '../lib/useCatalogFilters'

// Mirrors domain.ProductSorts. "Most loved" stays sales-based and rating is
// its own entry — see the note on DefaultProductSort for why an average over
// few reviews makes a bad default.
const SORTS: ProductSort[] = ['popular', 'rating', 'price_asc', 'price_desc', 'newest']

/**
 * The design's Shop screen: breadcrumbs, title, result count, sort, a
 * four-block sidebar and a paginated grid.
 *
 * Every filter lives in the query string (see useCatalogFilters), so the
 * back button undoes a filter, a reload keeps the view, and a pasted link
 * reproduces it exactly.
 *
 * The sidebar's counts come from their own query rather than being derived
 * from the products on screen — "Honey 1" has to be true across the whole
 * catalog, not across the twelve rows in the grid.
 */
export function ShopPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { filters, listParams, page, setFilter, toggleBenefit, clearAll, hasFilters } =
    useCatalogFilters()

  const products = useProducts(listParams)
  const facets = useCatalogFacets(filters)

  const me = useMe()
  const setCartItem = useSetCartItem()

  const total = products.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PER_PAGE))

  // The card's Add button buys the cheapest variant — the one whose price it
  // shows. Anything else would charge a price the visitor never saw. A
  // product with real choices is better bought on its own page, which is
  // where the variant picker lives (E3).
  const addToCart = (product: Product) => {
    const variant = product.variants.find((v) => v.stock_qty > 0)
    if (variant) setCartItem.mutate({ variantId: variant.id, qty: 1 })
  }

  const priceBounds = {
    min: facets.data?.price_min_minor ?? 0,
    max: facets.data?.price_max_minor ?? 0,
  }

  return (
    <div className="mx-auto max-w-360 px-6 py-9 lg:px-14">
      <Breadcrumbs
        items={[
          { label: t('common:nav.home'), to: localePath('/') },
          { label: t('common:nav.shop') },
        ]}
      />

      <div className="mt-3 flex flex-wrap items-end justify-between gap-6">
        <div className="flex flex-col gap-2.5">
          <h1 className="font-display text-display-lg font-extrabold text-ink">
            {t('catalog:shopTitle')}
          </h1>
          <p className="max-w-140 text-base text-ink-body">{t('catalog:shopBlurb')}</p>
        </div>

        <div className="flex items-center gap-3">
          {/* aria-live: the count changes without the page navigating, so a
              screen reader would otherwise never learn that a filter did
              anything. "polite" waits for a pause rather than interrupting. */}
          <p aria-live="polite" className="text-sm text-ink-muted">
            {t('common:productCount', { count: total })}
          </p>

          {/* A bare <select> rather than the <Select> primitive: that one
              renders a <label> above the box, and the design puts the label
              INSIDE the control ("Sort: Most loved"). The accessible name
              comes from aria-label instead, so the control is still named. */}
          <select
            aria-label={t('catalog:sortLabel')}
            value={filters.sort}
            onChange={(e) => setFilter({ sort: e.target.value })}
            className="rounded-full border-[1.5px] border-line bg-card px-5 py-2.5 font-display text-sm font-semibold text-ink"
          >
            {SORTS.map((s) => (
              <option key={s} value={s}>
                {t('catalog:sortPrefix')} {t(`catalog:sort.${s}`)}
              </option>
            ))}
          </select>
        </div>
      </div>

      <div className="mt-6 grid gap-8 lg:grid-cols-[260px_1fr]">
        <aside className="flex flex-col gap-5.5">
          <FilterBlock title={t('catalog:facet.category')}>
            <ul className="flex flex-col gap-2.5 text-base">
              <FacetRow
                label={t('catalog:allProducts')}
                count={facets.data?.total}
                active={!filters.category}
                onClick={() => setFilter({ category: undefined })}
              />
              {facets.data?.categories.map((c) => (
                <FacetRow
                  key={c.slug}
                  label={c.name}
                  count={c.count}
                  active={filters.category === c.slug}
                  onClick={() =>
                    setFilter({
                      category: filters.category === c.slug ? undefined : c.slug,
                    })
                  }
                />
              ))}
            </ul>
          </FilterBlock>

          {/* Hidden rather than rendered empty. A facet group with no rows
              means the search matched nothing, and three empty boxes above a
              "nothing matches" card read as a broken page rather than an
              answer. The blocks come back the moment anything matches. */}
          <FilterBlock
            title={t('catalog:facet.benefit')}
            hidden={facets.data?.benefits.length === 0}
          >
            <ul className="flex flex-wrap gap-2">
              {facets.data?.benefits.map((b) => {
                const active = filters.benefits?.includes(b.slug) ?? false
                return (
                  <li key={b.slug}>
                    <button
                      type="button"
                      // aria-pressed makes this a toggle rather than a
                      // button that happens to look different when on — a
                      // screen reader announces "Energy, pressed".
                      aria-pressed={active}
                      onClick={() => toggleBenefit(b.slug)}
                      className={cx(
                        'rounded-full border-[1.5px] px-3.5 py-2 text-xs transition',
                        active
                          ? 'border-honey bg-honey font-semibold text-ink'
                          : 'border-line text-ink-strong hover:border-line-strong',
                        b.count === 0 && !active && 'opacity-50',
                      )}
                    >
                      {b.name}
                      <span className="ml-1.5 text-ink-muted">{b.count}</span>
                    </button>
                  </li>
                )
              })}
            </ul>
          </FilterBlock>

          {/* `facets.data &&` matters: priceBounds falls back to 0 while the
              query is still in flight, and hiding on that would make the
              block flicker in on every load — "not loaded yet" is not the
              same state as "nothing matched". */}
          <FilterBlock
            title={t('catalog:facet.price')}
            hidden={!!facets.data && priceBounds.max === 0}
          >
            <PriceRange
              label={t('catalog:facet.price')}
              min={priceBounds.min}
              max={priceBounds.max}
              value={{
                min: filters.minPriceMinor ?? priceBounds.min,
                max: filters.maxPriceMinor ?? priceBounds.max,
              }}
              onCommit={({ min, max }) =>
                setFilter({
                  // Writing a bound equal to the catalog's own edge would be
                  // a filter that filters nothing, so it is dropped and the
                  // URL stays clean.
                  min_price: min > priceBounds.min ? String(min) : undefined,
                  max_price: max < priceBounds.max ? String(max) : undefined,
                })
              }
            />
          </FilterBlock>

          {hasFilters && (
            <Button variant="outline" size="sm" onClick={clearAll}>
              {t('catalog:clearFilters')}
            </Button>
          )}

          {/* "Not sure where to start?" — the design's dark help card. The
              button is inert until E9 builds a contact route. */}
          <Card tone="bark" className="flex flex-col gap-2.5">
            <h2 className="font-display text-lg font-bold text-ink-on-dark">
              {t('catalog:help.title')}
            </h2>
            <p className="text-sm leading-relaxed text-ink-on-dark-soft">
              {t('catalog:help.blurb')}
            </p>
            <Button variant="honey" fullWidth disabled className="mt-1">
              {t('catalog:help.cta')}
            </Button>
          </Card>
        </aside>

        <main className="flex flex-col gap-6.5">
          {products.isError && (
            <p className="rounded-lg bg-card p-4 text-danger">
              {t('common:state.loadFailed')}
            </p>
          )}

          {products.data && total === 0 && (
            <Card className="flex flex-col items-start gap-3 py-10">
              <p className="font-display text-lg font-bold text-ink">
                {t('catalog:noResults')}
              </p>
              <p className="text-sm text-ink-soft">{t('catalog:noResultsHint')}</p>
              {hasFilters && (
                <Button variant="outline" size="sm" onClick={clearAll}>
                  {t('catalog:clearFilters')}
                </Button>
              )}
            </Card>
          )}

          <div
            className={cx(
              'grid grid-cols-1 gap-5.5 sm:grid-cols-2 xl:grid-cols-3',
              // isPlaceholderData means we are showing the PREVIOUS page's
              // products while the next ones load. Dimming says "this is
              // stale" without the grid collapsing to a spinner and back.
              products.isPlaceholderData && 'opacity-60 transition-opacity',
            )}
          >
            {products.data?.items.map((p) => (
              <ProductCard key={p.id} product={p} onAdd={me.data ? addToCart : undefined} />
            ))}
          </div>

          {products.isPending && (
            <p className="text-ink-muted">{t('common:state.loading')}</p>
          )}

          {pageCount > 1 && (
            <Pagination
              page={page}
              pageCount={pageCount}
              onSelect={(n) => setFilter({ page: String(n) }, { resetPage: false })}
            />
          )}
        </main>
      </div>
    </div>
  )
}

function FilterBlock({
  title,
  hidden = false,
  children,
}: {
  title: string
  hidden?: boolean
  children: React.ReactNode
}) {
  if (hidden) return null
  return (
    <Card>
      <h2 className="mb-3.5 font-display text-base font-bold uppercase tracking-label text-ink">
        {title}
      </h2>
      {children}
    </Card>
  )
}

function FacetRow({
  label,
  count,
  active,
  onClick,
}: {
  label: string
  count?: number
  active: boolean
  onClick: () => void
}) {
  return (
    <li>
      <button
        type="button"
        aria-pressed={active}
        onClick={onClick}
        className={cx(
          'flex w-full items-center justify-between gap-3 text-left transition',
          active ? 'font-bold text-brand-ink' : 'text-ink-strong hover:text-ink',
        )}
      >
        <span>{label}</span>
        <span className={active ? undefined : 'text-ink-muted'}>{count ?? '—'}</span>
      </button>
    </li>
  )
}

function Pagination({
  page,
  pageCount,
  onSelect,
}: {
  page: number
  pageCount: number
  onSelect: (page: number) => void
}) {
  const { t } = useTranslation()
  const pages = Array.from({ length: pageCount }, (_, i) => i + 1)

  return (
    <nav aria-label={t('catalog:pagination')} className="flex justify-center gap-2.5 pt-2">
      {pages.map((n) => (
        <button
          key={n}
          type="button"
          aria-current={n === page ? 'page' : undefined}
          onClick={() => onSelect(n)}
          className={cx(
            'inline-flex size-9.5 items-center justify-center rounded-full font-display text-sm transition',
            n === page
              ? 'bg-brand-ink font-bold text-ink-on-dark'
              : 'border-[1.5px] border-line font-semibold text-ink-body hover:border-line-strong',
          )}
        >
          {n}
        </button>
      ))}
      <button
        type="button"
        aria-label={t('catalog:nextPage')}
        disabled={page >= pageCount}
        onClick={() => onSelect(page + 1)}
        className="inline-flex size-9.5 items-center justify-center rounded-full border-[1.5px] border-line text-sm text-ink-body transition hover:border-line-strong disabled:opacity-40"
      >
        →
      </button>
    </nav>
  )
}
