import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useCart, useProduct, useProducts } from './hooks'
import { CurrencyProvider } from '../lib/CurrencyProvider'
import { CurrencySwitcher } from '../components/ui/CurrencySwitcher'

/**
 * The E5 twin of locale.test.tsx, and it exists for the same reason: the
 * failure it guards against is INVISIBLE. A request with no ?currency= gets
 * a perfectly valid response — in dollars — so nothing errors, nothing looks
 * broken, and the shop is simply wrong for half its customers.
 *
 * It asserts on request URLs and on cache keys, because those are the two
 * places the bug can hide: sending the wrong currency, or sending the right
 * one and then rendering a cached answer to the previous question.
 */

let urls: string[] = []

beforeEach(() => {
  urls = []
  localStorage.clear()
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

afterEach(() => {
  vi.unstubAllGlobals()
  // The provider writes a cookie; clearing it keeps tests independent.
  document.cookie = 'mb_currency=; path=/; max-age=0'
})

function renderWithCurrency(ui: React.ReactNode, path = '/shop') {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={queryClient}>
        <CurrencyProvider>{ui}</CurrencyProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

function Probe({ useHook }: { useHook: () => unknown }) {
  useHook()
  return null
}

describe('catalog requests carry the active currency', () => {
  it('defaults to dollars with nothing stored', async () => {
    renderWithCurrency(<Probe useHook={() => useProducts({ page: 1 })} />)

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    expect(urls[0]).toContain('currency=USD')
  })

  it('reads a stored choice before the first request goes out', async () => {
    localStorage.setItem('mb_currency', 'AMD')
    renderWithCurrency(<Probe useHook={() => useProducts({ page: 1 })} />)

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    // Not a single dollar request: the provider sets the client's currency
    // during render, not in an effect, so there is no first-request window
    // where the shop briefly prices itself in the wrong market.
    expect(urls[0]).toContain('currency=AMD')
    expect(urls.filter((u) => u.includes('currency=USD'))).toHaveLength(0)
  })

  it('carries both the language and the currency on a detail read', async () => {
    localStorage.setItem('mb_currency', 'AMD')
    renderWithCurrency(<Probe useHook={() => useProduct('honey')} />, '/hy/products/honey')

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    expect(urls[0]).toBe('/api/v1/products/honey?lang=hy&currency=AMD')
  })

  it('carries the currency on the cart, which is priced too', async () => {
    localStorage.setItem('mb_currency', 'AMD')
    renderWithCurrency(<Probe useHook={() => useCart(true)} />)

    await waitFor(() => expect(urls.length).toBeGreaterThan(0))
    expect(urls[0]).toContain('currency=AMD')
  })
})

describe('switching currency', () => {
  it('refetches instead of serving the cached other-market prices', async () => {
    renderWithCurrency(
      <>
        <CurrencySwitcher />
        <Probe useHook={() => useProducts({ page: 1 })} />
      </>,
    )

    await waitFor(() => expect(urls.length).toBe(1))
    expect(urls[0]).toContain('currency=USD')

    fireEvent.click(screen.getByRole('radio', { name: /AMD/ }))

    // THE CACHE BUG THIS GUARDS: if the currency were not part of the query
    // key, the URL would change and TanStack Query would still hand back the
    // cached dollar response — no second request at all, and dollar prices
    // under a dram switcher.
    await waitFor(() => expect(urls.length).toBe(2))
    expect(urls[1]).toContain('currency=AMD')
  })

  it('remembers the choice for the next visit', async () => {
    renderWithCurrency(<CurrencySwitcher />)

    fireEvent.click(screen.getByRole('radio', { name: /AMD/ }))

    expect(localStorage.getItem('mb_currency')).toBe('AMD')
    // ...and tells the SERVER, which is what decides the checkout currency.
    await waitFor(() => expect(document.cookie).toContain('mb_currency=AMD'))
  })

  it('announces exactly one selected option', () => {
    renderWithCurrency(<CurrencySwitcher />)

    const group = screen.getByRole('radiogroup', { name: 'Currency' })
    expect(group).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /USD/ })).toHaveAttribute('aria-checked', 'true')
    expect(screen.getByRole('radio', { name: /AMD/ })).toHaveAttribute('aria-checked', 'false')
  })
})
