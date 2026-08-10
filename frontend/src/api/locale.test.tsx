import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useCategories, useProduct, useProducts } from './hooks'
import i18n from '../i18n'

/**
 * A regression suite for a bug E1.5 shipped and E2 found by LOOKING at the
 * running app: the backend served three languages, and the frontend never
 * asked for any of them. Every request went out with no ?lang=, no cookie
 * and no Accept-Language, so /hy/shop rendered an Armenian header around an
 * English catalog.
 *
 * Nothing failed. The backend's fallback chain returns valid English, so
 * there was no error to notice — which is exactly why this needs a test that
 * asserts on the URL rather than on whether the page renders.
 */

let urls: string[] = []

beforeEach(() => {
  urls = []
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL) => {
      urls.push(String(input))
      return Promise.resolve(
        new Response(JSON.stringify({ items: [], page: 1, per_page: 20, total: 0 }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    }),
  )
})

afterEach(async () => {
  vi.unstubAllGlobals()
  await i18n.changeLanguage('en')
})

function renderHookAt(path: string, useHook: () => unknown) {
  function Probe() {
    useHook()
    return null
  }
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={queryClient}>
        <Probe />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('catalog requests carry the active locale', () => {
  it('asks for Armenian on an /hy route', async () => {
    renderHookAt('/hy/shop', () => useProducts({ page: 1 }))

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    expect(urls[0]).toContain('lang=hy')
  })

  it('asks for English on an unprefixed route', async () => {
    renderHookAt('/shop', () => useProducts({ page: 1 }))

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    expect(urls[0]).toContain('lang=en')
  })

  it('sets the locale during render, not in an effect', async () => {
    // The distinction that matters: an effect runs AFTER the query fires, so
    // the very first request on a fresh page load would ask for English and
    // only a refetch would be right.
    renderHookAt('/ru', () => useCategories())

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    expect(urls[0]).toContain('lang=ru')
    expect(urls.filter((u) => u.includes('lang=en'))).toHaveLength(0)
  })

  it('appends lang to a URL that already has a query string', async () => {
    renderHookAt('/hy/shop', () => useProducts({ category: 'honey', page: 2 }))

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    expect(urls[0]).toMatch(/category=honey.*&lang=hy/)
  })

  it('carries the locale on a product detail read', async () => {
    renderHookAt('/ru/products/honey', () => useProduct('honey'))

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    expect(urls[0]).toBe('/api/v1/products/honey?lang=ru')
  })
})
