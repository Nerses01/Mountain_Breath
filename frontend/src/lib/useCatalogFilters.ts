import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router'
import type { CatalogFilterParams, ProductListParams } from '../api/client'
import type { ProductSort } from '../api/types'

export const PER_PAGE = 12

// Mirrors domain.ProductSorts. A value here that the backend does not
// whitelist would be a URL that silently falls back to popular; a value the
// backend has and this list does not would be a link nobody can reach.
const SORTS: readonly ProductSort[] = [
  'popular',
  'rating',
  'price_asc',
  'price_desc',
  'newest',
]

function parseSort(raw: string | null): ProductSort {
  return SORTS.includes(raw as ProductSort) ? (raw as ProductSort) : 'popular'
}

function parseMinor(raw: string | null): number | undefined {
  if (raw === null) return undefined
  const n = Number(raw)
  return Number.isFinite(n) && n >= 0 ? n : undefined
}

/**
 * All Shop-page filter state, read from and written to the QUERY STRING.
 *
 * No useState anywhere in the shop's filters, deliberately. React state
 * would be faster to write and would break four things at once: the back
 * button (a filter click leaves no history entry to go back to), a shared
 * link (the recipient sees the unfiltered shop), a reload (everything
 * resets), and opening a result in a new tab. The URL already is a place to
 * put state, and it is the only one the browser knows how to navigate.
 *
 * The C++ instinct here is to cache the parsed value in a member and keep it
 * in sync. Resist it: the search params ARE the state, parsed on every
 * render. There is no second copy, so there is nothing to fall out of sync —
 * the same reason the store resolves a product's name from the database
 * rather than keeping a denormalized copy per locale.
 */
export function useCatalogFilters() {
  const [params, setParams] = useSearchParams()

  // `& { sort: ... }`: parseSort always supplies a default, so unlike the
  // request params — where sort is optional — the PARSED filters always
  // carry one. Saying so in the type is what lets the sort control take a
  // value instead of a maybe.
  const filters: CatalogFilterParams & { sort: ProductSort } = useMemo(
    () => ({
      category: params.get('category') ?? undefined,
      q: params.get('q') ?? undefined,
      // getAll, not get: several chips arrive as repeated params, exactly as
      // the API reads them.
      benefits: params.getAll('benefit'),
      minPriceMinor: parseMinor(params.get('min_price')),
      maxPriceMinor: parseMinor(params.get('max_price')),
      sort: parseSort(params.get('sort')),
    }),
    [params],
  )

  const page = Math.max(1, Number(params.get('page') ?? 1) || 1)

  const listParams: ProductListParams = useMemo(
    () => ({ ...filters, page, perPage: PER_PAGE }),
    [filters, page],
  )

  /**
   * Writes one or more params, dropping any whose value is empty so the URL
   * stays as short as what is actually selected — `/shop` rather than
   * `/shop?category=&q=&sort=popular`.
   *
   * `resetPage` defaults to true because narrowing a filter almost always
   * invalidates the page number: filtering to two products while on page 3
   * shows an empty grid, which reads as a broken shop rather than as a page
   * that no longer exists. The pagination control passes false.
   */
  const setFilter = useCallback(
    (
      updates: Record<string, string | string[] | undefined>,
      { resetPage = true }: { resetPage?: boolean } = {},
    ) => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          for (const [key, value] of Object.entries(updates)) {
            next.delete(key)
            if (Array.isArray(value)) {
              for (const v of value) next.append(key, v)
            } else if (value) {
              next.set(key, value)
            }
          }
          if (resetPage) next.delete('page')
          return next
        },
        // replace: false — every filter click IS a history entry, which is
        // what makes the back button undo a filter.
        { replace: false },
      )
    },
    [setParams],
  )

  const toggleBenefit = useCallback(
    (slug: string) => {
      const current = params.getAll('benefit')
      setFilter({
        benefit: current.includes(slug)
          ? current.filter((s) => s !== slug)
          : [...current, slug],
      })
    },
    [params, setFilter],
  )

  const clearAll = useCallback(() => setParams(new URLSearchParams()), [setParams])

  const hasFilters =
    Boolean(filters.category) ||
    Boolean(filters.q) ||
    (filters.benefits?.length ?? 0) > 0 ||
    filters.minPriceMinor !== undefined ||
    filters.maxPriceMinor !== undefined

  return { filters, listParams, page, setFilter, toggleBenefit, clearAll, hasFilters }
}
