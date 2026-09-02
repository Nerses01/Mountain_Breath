import { describe, expect, it } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { MemoryRouter, useLocation } from 'react-router'
import { useCatalogFilters } from './useCatalogFilters'

/**
 * The filter state lives in the query string, which is a claim worth testing
 * rather than trusting: it is what makes the back button undo a filter and a
 * pasted link reproduce a view. A useState implementation would pass every
 * "does clicking work" test and fail all of these.
 */

// A probe component: renders the parsed filter and the live URL as text, and
// exposes the setters through a ref-like object the test can call.
let api: ReturnType<typeof useCatalogFilters>

function Probe() {
  api = useCatalogFilters()
  const { search } = useLocation()
  return (
    <>
      <output data-testid="search">{search}</output>
      <output data-testid="filters">{JSON.stringify(api.filters)}</output>
      <output data-testid="page">{api.page}</output>
      <output data-testid="hasFilters">{String(api.hasFilters)}</output>
    </>
  )
}

function renderAt(url: string) {
  return render(
    <MemoryRouter initialEntries={[url]}>
      <Probe />
    </MemoryRouter>,
  )
}

const url = () => screen.getByTestId('search').textContent
const filters = () => JSON.parse(screen.getByTestId('filters').textContent ?? '{}')

describe('useCatalogFilters', () => {
  it('reads every filter out of the query string', () => {
    renderAt('/shop?category=honey&q=jelly&benefit=energy&benefit=skin&min_price=900&max_price=3200&sort=price_asc&page=2')

    expect(filters()).toMatchObject({
      category: 'honey',
      q: 'jelly',
      benefits: ['energy', 'skin'],
      minPriceMinor: 900,
      maxPriceMinor: 3200,
      sort: 'price_asc',
    })
    expect(screen.getByTestId('page').textContent).toBe('2')
  })

  it('defaults to the popular sort and no filters', () => {
    renderAt('/shop')

    expect(filters()).toMatchObject({ sort: 'popular', benefits: [] })
    expect(screen.getByTestId('hasFilters').textContent).toBe('false')
    expect(screen.getByTestId('page').textContent).toBe('1')
  })

  it('falls back on a sort the API does not offer', () => {
    // A hand-edited URL must render the shop, not an error — the backend
    // whitelists the same way, so the two agree about what "popular" means.
    renderAt('/shop?sort=cheapest')

    expect(filters().sort).toBe('popular')
  })

  it('ignores a negative or unparseable price bound', () => {
    renderAt('/shop?min_price=-500&max_price=cheap')

    expect(filters().minPriceMinor).toBeUndefined()
    expect(filters().maxPriceMinor).toBeUndefined()
  })

  it('writes a filter into the URL', () => {
    renderAt('/shop')

    act(() => api.setFilter({ category: 'honey' }))

    expect(url()).toBe('?category=honey')
    expect(filters().category).toBe('honey')
  })

  it('drops an empty value instead of leaving a bare key behind', () => {
    renderAt('/shop?category=honey')

    act(() => api.setFilter({ category: undefined }))

    // `?category=` would still be a param, and a URL full of empty keys is
    // unshareable noise.
    expect(url()).toBe('')
  })

  it('toggles a benefit chip on and off, keeping the others', () => {
    renderAt('/shop?benefit=energy')

    act(() => api.toggleBenefit('skin'))
    expect(filters().benefits).toEqual(['energy', 'skin'])
    expect(url()).toBe('?benefit=energy&benefit=skin')

    act(() => api.toggleBenefit('energy'))
    expect(filters().benefits).toEqual(['skin'])
  })

  it('resets the page when a filter narrows the result', () => {
    renderAt('/shop?category=honey&page=3')

    act(() => api.setFilter({ benefit: ['energy'] }))

    // Filtering to two products while on page 3 would show an empty grid,
    // which reads as a broken shop rather than a page that no longer exists.
    expect(url()).not.toContain('page=3')
    expect(screen.getByTestId('page').textContent).toBe('1')
  })

  it('keeps the page when the pagination control moves it', () => {
    renderAt('/shop?category=honey')

    act(() => api.setFilter({ page: '2' }, { resetPage: false }))

    expect(screen.getByTestId('page').textContent).toBe('2')
    expect(filters().category).toBe('honey')
  })

  it('clears everything at once', () => {
    renderAt('/shop?category=honey&benefit=energy&sort=newest&page=2')

    act(() => api.clearAll())

    expect(url()).toBe('')
    expect(screen.getByTestId('hasFilters').textContent).toBe('false')
  })

  it('does not count the sort as a filter', () => {
    // Sorting reorders the same products; there is nothing to "clear", so
    // the Clear filters button must not appear for it alone.
    renderAt('/shop?sort=newest')

    expect(screen.getByTestId('hasFilters').textContent).toBe('false')
  })
})
