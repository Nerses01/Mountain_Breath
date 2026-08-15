import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { WishlistHeart } from './WishlistHeart'

/**
 * The heart's contract: state derives from the wishlist QUERY (so every
 * heart for one product agrees), toggling states the DESIRED state to the
 * API, and a signed-out visitor gets a disabled control, not a trap.
 */

function stubFetch(handlers: Record<string, () => Response>) {
  const calls: { url: string; method: string }[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      const method = init?.method ?? 'GET'
      calls.push({ url, method })
      for (const [needle, make] of Object.entries(handlers)) {
        if (url.includes(needle)) return Promise.resolve(make())
      }
      return Promise.resolve(
        new Response('{}', { status: 200, headers: { 'Content-Type': 'application/json' } }),
      )
    }),
  )
  return calls
}

const me = () =>
  new Response(
    JSON.stringify({
      id: 1, email: 'a@x', role: 'customer',
      hive: { prior_orders: 0, member: false, member_discount_percent: 0, first_delivery_free: true },
    }),
    { status: 200, headers: { 'Content-Type': 'application/json' } },
  )

function renderHeart(productId: number) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <WishlistHeart productId={productId} />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

afterEach(() => vi.unstubAllGlobals())

describe('WishlistHeart', () => {
  it('derives its state from the wishlist and toggles by stating the opposite', async () => {
    const calls = stubFetch({
      '/auth/me': me,
      '/wishlist': () =>
        new Response(JSON.stringify([{ id: 5 }]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
    })
    renderHeart(5)

    const heart = await screen.findByRole('button', { name: 'Wishlist' })
    await waitFor(() => expect(heart).toHaveAttribute('aria-pressed', 'true'))

    fireEvent.click(heart)
    await waitFor(() =>
      expect(calls).toContainEqual(
        expect.objectContaining({ method: 'DELETE', url: expect.stringContaining('/wishlist/5') }),
      ),
    )
  })

  it('is disabled with the reason for signed-out visitors', async () => {
    stubFetch({
      '/auth/me': () =>
        new Response(JSON.stringify({ error: { code: 'unauthorized', message: 'no' } }), {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        }),
    })
    renderHeart(5)

    const heart = await screen.findByRole('button', { name: 'Wishlist' })
    await waitFor(() => expect(heart).toBeDisabled())
    expect(heart).toHaveAttribute('title', 'Sign in to save products')
  })
})
