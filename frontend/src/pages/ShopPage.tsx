import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useCatalogFacets, useProducts, useQuickAdd } from '../api/hooks'
import type { ProductSort } from '../api/types'
import { ProductCard } from '../components/ProductCard'
import {
  Breadcrumbs,
  Button,
  Card,
  IconButton,
  Pagination,
  PillSelect,
  XIcon,
} from '../components/ui'
import { PriceRange } from '../components/ui/PriceRange'
import { useLocale } from '../i18n/useLocale'
import { cx } from '../lib/cx'
import { usePageMeta } from '../lib/usePageMeta'
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

  // undefined while signed out, which the card renders as a disabled Add.
  const quickAdd = useQuickAdd()

  // E10 SEO: whatever filters are in the query string, the canonical names
  // the CLEAN /shop — a thousand filter permutations must not compete with
  // the one page in a search index.
  usePageMeta({ title: t('catalog:shopTitle'), canonicalPath: '/shop' })

  const total = products.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PER_PAGE))

  // E10: the drawer that carries the sidebar below lg. A real dialog while
  // open — Escape closes, focus lands on its close button and returns to
  // the Filters button after — because on a phone it genuinely covers the
  // page, unlike the header's in-flow disclosure sheet.
  const [drawerOpen, setDrawerOpen] = useState(false)
  const filterButtonRef = useRef<HTMLButtonElement>(null)
  const drawerCloseRef = useRef<HTMLButtonElement>(null)
  const closeDrawer = () => {
    setDrawerOpen(false)
    filterButtonRef.current?.focus()
  }
  useEffect(() => {
    if (!drawerOpen) return
    drawerCloseRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeDrawer()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [drawerOpen])

  // The button's badge: how many filter GROUPS are active, so a closed
  // drawer still says whether it is hiding decisions.
  const activeFilterCount =
    (filters.category ? 1 : 0) +
    (filters.benefits?.length ?? 0) +
    (filters.minPriceMinor !== undefined || filters.maxPriceMinor !== undefined ? 1 : 0)

  const priceBounds = {
    min: facets.data?.price_min_minor ?? 0,
    max: facets.data?.price_max_minor ?? 0,
  }

  // The whole filter column as one JSX value, mounted in the desktop aside
  // AND the mobile drawer — one source for the filters, two frames.
  const filterPanel = (
    <>
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

      {/* "Not sure where to start?" — the design's dark help card, live
          since E9 gave it a contact page to point at. */}
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
    </>
  )

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
          <h1 className="font-display text-display-md font-extrabold text-ink lg:text-display-lg">
            {t('catalog:shopTitle')}
          </h1>
          <p className="max-w-140 text-base text-ink-body">{t('catalog:shopBlurb')}</p>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          {/* aria-live: the count changes without the page navigating, so a
              screen reader would otherwise never learn that a filter did
              anything. "polite" waits for a pause rather than interrupting. */}
          <p aria-live="polite" className="text-sm text-ink-muted">
            {t('common:productCount', { count: total })}
          </p>

          <button
            type="button"
            aria-expanded={drawerOpen}
            aria-controls="shop-filters"
            onClick={() => setDrawerOpen(true)}
            ref={filterButtonRef}
            className="rounded-full border-[1.5px] border-line bg-card px-5 py-2.5 font-display text-sm font-semibold text-ink lg:hidden"
          >
            {t('catalog:filters')}
            {activeFilterCount > 0 && (
              <span className="ml-1.5 text-ink-muted">· {activeFilterCount}</span>
            )}
          </button>

          {/* PillSelect, not a native <select>: the OS draws a select's open
              list and takes no CSS, and the design wants the popup on its
              own palette. The prefix lives on the CLOSED pill only — inside
              the list the rows say "Most loved", not "Sort: Most loved"
              five times over. */}
          <PillSelect
            ariaLabel={t('catalog:sortLabel')}
            prefix={t('catalog:sortPrefix')}
            value={filters.sort}
            onChange={(sort) => setFilter({ sort })}
            options={SORTS.map((s) => ({ value: s, label: t(`catalog:sort.${s}`) }))}
          />
        </div>
      </div>

      <div className="mt-6 grid gap-8 lg:grid-cols-[260px_1fr]">
        {/* E10: below lg the sidebar becomes a drawer (the plan's 1024
            breakpoint) — four filter blocks stacked ABOVE the grid pushed
            every product below the fold. The aside hides, the Filters
            button appears, and the same JSX renders in both places
            (PriceRange's useId keeps the double mount collision-free). */}
        <aside className="hidden flex-col gap-5.5 lg:flex">{filterPanel}</aside>

        {drawerOpen && (
          <div
            className="fixed inset-0 z-50 flex bg-bark/40 lg:hidden"
            onClick={(e) => {
              if (e.target === e.currentTarget) closeDrawer()
            }}
          >
            <div
              id="shop-filters"
              role="dialog"
              aria-modal="true"
              aria-label={t('catalog:filters')}
              className="flex h-full w-80 max-w-[85vw] flex-col gap-5.5 overflow-y-auto bg-page p-6"
            >
              <div className="flex items-center justify-between">
                <h2 className="font-display text-lg font-bold text-ink">
                  {t('catalog:filters')}
                </h2>
                <IconButton ref={drawerCloseRef} label={t('common:actions.close')} onClick={closeDrawer}>
                  <XIcon />
                </IconButton>
              </div>
              {filterPanel}
            </div>
          </div>
        )}

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
              <ProductCard key={p.id} product={p} onAdd={quickAdd} />
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

// Pagination moved to components/ui — the order history pages with it too.
