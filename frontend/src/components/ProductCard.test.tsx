import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ProductCard } from './ProductCard'
import type { Product } from '../api/types'

const product: Product = {
  id: 1,
  category_id: 1,
  category_slug: 'honey',
  category_name: 'Honey',
  rating_avg: 4.67,
  rating_count: 3,
  slug: 'mountain-wildflower-honey',
  name: 'Mountain Wildflower Honey',
  description: 'Sweet liquid made from flower nectar.',
  image_url: '',
  created_at: '2026-07-29T00:00:00Z',
  badge: 'best_seller',
  badge_tone: 'honey',
  benefits: [
    { slug: 'energy', name: 'Energy' },
    { slug: 'sweetening', name: 'Sweetening' },
  ],
  currency: 'USD',
  variants: [
    {
      id: 1,
      sku: 'HON-500',
      label: '500 g',
      price_minor: 1400,
      prices: { USD: 1400, AMD: 6700 },
      stock_qty: 40,
    },
    {
      id: 2,
      sku: 'HON-1K',
      label: '1 kg',
      price_minor: 2600,
      prices: { USD: 2600, AMD: 12400 },
      stock_qty: 6,
    },
  ],
}

// Components using <Link> need a router context — MemoryRouter is the
// test-friendly one (no real browser URL involved). E8's live heart added
// the query provider: the card now asks who is signed in.
function renderCard(p: Product, onAdd?: (p: Product) => void) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <ProductCard product={p} onAdd={onAdd} />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('ProductCard', () => {
  it('shows the name, the "size · benefit" line and the from-price', () => {
    renderCard(product)

    expect(screen.getByText('Mountain Wildflower Honey')).toBeInTheDocument()
    // The eyebrow is the CATEGORY, resolved server-side into the reader's
    // language; the line under the name pairs the size with the product's
    // first benefit by the taxonomy's sort_order.
    expect(screen.getByText('Honey')).toBeInTheDocument()
    expect(screen.getByText('500 g · Energy')).toBeInTheDocument()
    // Two variants, so the price is labelled "from" — an unlabelled $14 on a
    // product that also sells for $26 would be a lie of omission.
    expect(screen.getByText('from $14.00')).toBeInTheDocument()
    // ...and the design's muted second line, which is a SHELF price from the
    // other market, not $14.00 run through a rate.
    expect(screen.getByText('6,700 ֏')).toBeInTheDocument()
  })

  it('shows only one price when the shop has none in the other market', () => {
    renderCard({
      ...product,
      variants: [{ ...product.variants[0], prices: { USD: 1400 } }],
    })

    expect(screen.getByText('$14.00')).toBeInTheDocument()
    expect(screen.queryByText(/֏/)).not.toBeInTheDocument()
  })

  it('renders the badge KEY through the message catalogue, not raw', () => {
    renderCard(product)

    expect(screen.getByText('Best seller')).toBeInTheDocument()
    expect(screen.queryByText('best_seller')).not.toBeInTheDocument()
  })

  it('omits the badge when the product has none', () => {
    renderCard({ ...product, badge: '' })

    expect(screen.queryByText('Best seller')).not.toBeInTheDocument()
  })

  it('drops the "from" label when there is only one size', () => {
    renderCard({ ...product, variants: [product.variants[0]] })

    expect(screen.getByText('$14.00')).toBeInTheDocument()
    expect(screen.queryByText('from $14.00')).not.toBeInTheDocument()
  })

  it('marks a product with no stock in any variant as out of stock', () => {
    const soldOut = {
      ...product,
      variants: product.variants.map((v) => ({ ...v, stock_qty: 0 })),
    }
    renderCard(soldOut)

    expect(screen.getByText('Out of stock')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add' })).toBeDisabled()
  })

  it('stays in stock while ANY variant has some', () => {
    const partial = {
      ...product,
      variants: [
        { ...product.variants[0], stock_qty: 0 },
        { ...product.variants[1], stock_qty: 3 },
      ],
    }
    renderCard(partial, () => {})

    expect(screen.queryByText('Out of stock')).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Add' })).toBeEnabled()
  })

  it('hands the product to the Add handler', () => {
    const onAdd = vi.fn()
    renderCard(product, onAdd)

    fireEvent.click(screen.getByRole('button', { name: 'Add' }))

    expect(onAdd).toHaveBeenCalledWith(product)
  })

  it('disables Add when no handler is wired (anonymous visitor)', () => {
    renderCard(product)

    expect(screen.getByRole('button', { name: 'Add' })).toBeDisabled()
  })

  it('links the name to the product page', () => {
    renderCard(product)

    expect(
      screen.getByRole('link', { name: 'Mountain Wildflower Honey' }),
    ).toHaveAttribute('href', '/products/mountain-wildflower-honey')
  })
})
