import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { ProductCard } from './ProductCard'
import type { Product } from '../api/types'

const product: Product = {
  id: 1,
  category_id: 1,
  slug: 'wild-thyme-tea',
  name: 'Wild Thyme Tea',
  description: 'Fragrant wild thyme.',
  image_url: '',
  created_at: '2026-07-29T00:00:00Z',
  variants: [
    { id: 1, sku: 'TEA-1', label: '50 g', price_minor: 120000, stock_qty: 40 },
    { id: 2, sku: 'TEA-2', label: '100 g', price_minor: 220000, stock_qty: 0 },
  ],
}

// Components using <Link> need a router context — MemoryRouter is the
// test-friendly one (no real browser URL involved).
function renderCard(p: Product) {
  return render(
    <MemoryRouter>
      <ProductCard product={p} />
    </MemoryRouter>,
  )
}

describe('ProductCard', () => {
  it('shows name, description, and both variants', () => {
    renderCard(product)

    expect(screen.getByText('Wild Thyme Tea')).toBeInTheDocument()
    expect(screen.getByText('Fragrant wild thyme.')).toBeInTheDocument()
    expect(screen.getByText('50 g')).toBeInTheDocument()
    expect(screen.getByText('100 g')).toBeInTheDocument()
  })

  it('formats prices from minor units', () => {
    renderCard(product)

    expect(screen.getByText('1,200.00')).toBeInTheDocument()
    expect(screen.getByText('2,200.00')).toBeInTheDocument()
  })

  it('marks stock state per variant', () => {
    renderCard(product)

    expect(screen.getByText('40 left')).toBeInTheDocument()
    expect(screen.getByText('out of stock')).toBeInTheDocument()
  })

  it('links to the product page', () => {
    renderCard(product)

    expect(screen.getByRole('link')).toHaveAttribute('href', '/products/wild-thyme-tea')
  })
})
