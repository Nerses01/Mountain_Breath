import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router'
import { OrderSummary } from './OrderSummary'
import { PromoBox } from './PromoBox'
import { FreeShippingBanner } from './FreeShippingBanner'
import type { Cart, Preview } from '../../api/types'

/**
 * The E7 money components. What these pin:
 *
 *  - the summary's discount rows exist exactly when the server granted a
 *    discount — no permanent "− $0.00" noise, no hidden discount;
 *  - free shipping is labelled with its REASON (the first-order perk by
 *    name), because a kept promise deserves naming;
 *  - the promo box renders the server's validation code as a sentence, and
 *    an applied-but-dead code complains by name instead of vanishing;
 *  - the banner's three states, and its absence when there is no threshold
 *    to count toward.
 */

const cart: Cart = {
  items: [
    {
      variant_id: 1, product_name: 'Mountain Wildflower Honey', product_slug: 'honey',
      label: '500 g', stock_qty: 10, qty: 2,
      price_minor: 1400, line_total_minor: 2800,
      prices: { USD: 1400, AMD: 6700 }, line_totals: { USD: 2800, AMD: 13400 },
    },
  ],
  currency: 'USD',
  subtotal_minor: 2800,
  has_cold_chain: false,
  subtotals: { USD: 2800, AMD: 13400 },
}

function preview(overrides: Partial<Preview> = {}): Preview {
  return {
    currency: 'USD',
    subtotal_minor: 2800,
    shipping_minor: 400,
    member_discount_minor: 0,
    promo_discount_minor: 0,
    discount_minor: 0,
    tax_minor: 467,
    total_minor: 3200,
    has_cold_chain: false,
    first_delivery_free: false,
    base_shipping_waived: false,
    totals: { USD: 3200, AMD: 15300 },
    ...overrides,
  }
}

function wrap(ui: React.ReactElement) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>{ui}</QueryClientProvider>
    </MemoryRouter>,
  )
}

afterEach(() => vi.unstubAllGlobals())

describe('OrderSummary', () => {
  it('draws both discount lines when the server granted them', () => {
    wrap(
      <OrderSummary
        cart={cart}
        preview={preview({
          member_discount_minor: 224,
          promo_discount_minor: 280,
          promo_code: 'HONEY10',
          discount_minor: 504,
          total_minor: 2696,
        })}
      />,
    )

    expect(screen.getByText('Hive club discount')).toBeInTheDocument()
    expect(screen.getByText('− $2.24')).toBeInTheDocument()
    expect(screen.getByText('Code HONEY10')).toBeInTheDocument()
    expect(screen.getByText('− $2.80')).toBeInTheDocument()
    expect(screen.getByText('$26.96')).toBeInTheDocument()
  })

  it('hides discount rows that did not happen', () => {
    wrap(<OrderSummary cart={cart} preview={preview()} />)
    expect(screen.queryByText('Hive club discount')).not.toBeInTheDocument()
    expect(screen.queryByText(/^Code /)).not.toBeInTheDocument()
  })

  it('names the first-order perk when it is why shipping is free', () => {
    wrap(
      <OrderSummary
        cart={cart}
        preview={preview({
          shipping_minor: 0,
          first_delivery_free: true,
          base_shipping_waived: true,
          total_minor: 2800,
        })}
      />,
    )
    expect(screen.getByText('Free — first order')).toBeInTheDocument()
  })
})

describe('PromoBox', () => {
  it('renders the server’s refusal code as a sentence under the input', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(() =>
        Promise.resolve(
          new Response(
            JSON.stringify({
              error: {
                code: 'validation_failed',
                message: 'validation failed',
                fields: { promo_code: 'promo_expired' },
              },
            }),
            { status: 400, headers: { 'Content-Type': 'application/json' } },
          ),
        ),
      ),
    )

    wrap(<PromoBox preview={preview()} />)
    fireEvent.change(screen.getByLabelText('Promo code'), { target: { value: 'AUGUST' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply' }))

    // The catalogue's sentence, not the raw code.
    expect(await screen.findByRole('alert')).toHaveTextContent('This code has expired')
  })

  it('shows an applied code with its remove control, and its issue by name', () => {
    wrap(
      <PromoBox
        preview={preview({ promo_code: 'HONEY10', promo_issue: 'promo_min_subtotal' })}
      />,
    )
    expect(screen.getByText('HONEY10 applied')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Remove code' })).toBeInTheDocument()
    // The basket shrank below the floor since applying: the box says so
    // rather than the discount silently disappearing from the numbers.
    expect(screen.getByRole('alert')).toHaveTextContent('This code needs a larger basket')
  })
})

describe('FreeShippingBanner', () => {
  it('counts down, fills the bar, and offers the gap-closing product', () => {
    wrap(
      <FreeShippingBanner
        cart={cart}
        preview={preview({
          free_shipping_threshold_minor: 7000,
          free_shipping_remaining_minor: 4200,
          upsell: { variant_id: 9, slug: 'bee-pollen-granules', name: 'Bee Pollen Granules', price_minor: 1600 },
        })}
      />,
    )

    expect(screen.getByText('$42.00 away from free shipping')).toBeInTheDocument()
    // 2800 of 7000 = 40%.
    expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '40')
    expect(
      screen.getByRole('button', { name: 'Add Bee Pollen Granules · $16.00' }),
    ).toBeInTheDocument()
  })

  it('celebrates the unlocked state instead of counting to zero', () => {
    wrap(
      <FreeShippingBanner
        cart={cart}
        preview={preview({ shipping_minor: 0, base_shipping_waived: true })}
      />,
    )
    expect(screen.getByText('Free shipping unlocked')).toBeInTheDocument()
    expect(screen.queryByRole('progressbar')).not.toBeInTheDocument()
  })

  it('renders nothing when the market has no threshold', () => {
    const { container } = wrap(<FreeShippingBanner cart={cart} preview={preview()} />)
    expect(container).toBeEmptyDOMElement()
  })
})
