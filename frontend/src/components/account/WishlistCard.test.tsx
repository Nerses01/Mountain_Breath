import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { WishlistCard } from './WishlistCard'
import type { WishlistEntry } from '../../api/types'

/**
 * The wishlist card's contract (A3): the saved-ago line renders in words,
 * the price is the card's own "from" price, Add follows stock, and sold-out
 * shows its label WITHOUT a decorative "Notify me" (decision #6 deferral).
 */

function entry(over: Partial<WishlistEntry> = {}): WishlistEntry {
  return {
    id: 5,
    category_id: 1,
    category_slug: 'honey',
    category_name: 'Honey',
    rating_avg: 0,
    rating_count: 0,
    slug: 'royal-jelly',
    name: 'Fresh Royal Jelly',
    description: '',
    image_url: '',
    created_at: '2026-08-01T00:00:00Z',
    variants: [
      { id: 9, sku: 'RJ-50', label: '50 g jar', stock_qty: 4, price_minor: 5800, prices: { USD: 5800 } },
    ],
    badge: '',
    badge_tone: 'honey',
    benefits: [],
    currency: 'USD',
    // 14 days before the frozen "now" below.
    saved_at: '2026-08-04T12:00:00Z',
    ...over,
  }
}

beforeEach(() => {
  vi.useFakeTimers({ now: new Date('2026-08-18T12:00:00Z'), toFake: ['Date'] })
  vi.stubGlobal(
    'fetch',
    vi.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify([]), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    ),
  )
})
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

function renderCard(e: WishlistEntry, onAdd?: () => void) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <WishlistCard entry={e} onAdd={onAdd} />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('WishlistCard', () => {
  it('shows size and the saved-ago line in words', () => {
    renderCard(entry(), vi.fn())
    // 14 days → "2 weeks ago" via Intl.RelativeTimeFormat.
    expect(screen.getByText(/50 g jar · saved 2 weeks ago/)).toBeInTheDocument()
  })

  it('in stock: Add to cart is enabled and no sold-out label', () => {
    renderCard(entry(), vi.fn())
    expect(screen.getByRole('button', { name: 'Add to cart' })).toBeEnabled()
    expect(screen.queryByText('Out of stock')).not.toBeInTheDocument()
  })

  it('sold out: label shows, Add disables, and there is NO Notify me', () => {
    renderCard(
      entry({
        variants: [
          { id: 9, sku: 'RJ-50', label: '50 g jar', stock_qty: 0, price_minor: 5800, prices: { USD: 5800 } },
        ],
      }),
      vi.fn(),
    )
    expect(screen.getByText('Out of stock')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add to cart' })).toBeDisabled()
    expect(screen.queryByText(/notify/i)).not.toBeInTheDocument()
  })

  it('several variants: the price line reads as "from"', () => {
    renderCard(
      entry({
        variants: [
          { id: 9, sku: 'RJ-25', label: '25 g jar', stock_qty: 4, price_minor: 3200, prices: { USD: 3200 } },
          { id: 10, sku: 'RJ-50', label: '50 g jar', stock_qty: 4, price_minor: 5800, prices: { USD: 5800 } },
        ],
      }),
      vi.fn(),
    )
    expect(screen.getByText(/from \$32\.00/)).toBeInTheDocument()
  })
})
