import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CheckoutPage } from './CheckoutPage'
import type { Cart } from '../api/types'

/**
 * The checkout's three contracts, tested at the component boundary:
 *
 *  1. an empty form fails CLIENT-side — no request leaves the page;
 *  2. a filled form posts exactly the CheckoutInput shape, and no money;
 *  3. the design's "AMD only" cash rule renders as a disabled option, not
 *     as a surprise rejection after submit.
 *
 * fetch is stubbed at the network edge (the same seam locale.test.tsx uses)
 * so the whole real stack — client, hooks, provider — runs in between.
 */

const cartFixture: Cart = {
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
  shipping_minor: 400,
  total_minor: 3200,
  has_cold_chain: false,
  subtotals: { USD: 2800, AMD: 13400 },
  shipping: { USD: 400, AMD: 1900 },
  totals: { USD: 3200, AMD: 15300 },
}

let requests: { url: string; body: unknown }[] = []

beforeEach(() => {
  requests = []
  localStorage.clear()
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      requests.push({ url, body: init?.body ? JSON.parse(String(init.body)) : undefined })

      let payload: unknown = {}
      if (url.includes('/auth/me')) payload = { id: 1, email: 'anahit@example.com', role: 'customer' }
      else if (url.includes('/cart')) payload = cartFixture
      else if (url.includes('/account/address')) {
        return Promise.resolve(
          new Response(JSON.stringify({ error: { code: 'not_found', message: 'no saved address' } }), {
            status: 404, headers: { 'Content-Type': 'application/json' },
          }),
        )
      } else if (url.includes('/orders')) payload = { id: 42, status: 'pending', currency: 'USD', items: [] }

      return Promise.resolve(
        new Response(JSON.stringify(payload), {
          status: url.includes('/orders') ? 201 : 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    }),
  )
})

afterEach(() => {
  vi.unstubAllGlobals()
  document.cookie = 'mb_currency=; path=/; max-age=0'
})

function renderCheckout() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter initialEntries={['/checkout']}>
      <QueryClientProvider client={qc}>
        <Routes>
          <Route path="/checkout" element={<CheckoutPage />} />
          <Route path="/orders/:id" element={<p>order page</p>} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

async function settle() {
  await screen.findByRole('heading', { name: 'Where should the jars go?' })
}

function fillAddress() {
  fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Anahit' } })
  fireEvent.change(screen.getByLabelText('Last name'), { target: { value: 'Sargsyan' } })
  fireEvent.change(screen.getByLabelText('Phone'), { target: { value: '+374 91 000000' } })
  fireEvent.change(screen.getByLabelText('Street and number'), { target: { value: '14 Abovyan St, apt 6' } })
  fireEvent.change(screen.getByLabelText('City'), { target: { value: 'Yerevan' } })
  fireEvent.change(screen.getByLabelText('Postal code'), { target: { value: '0009' } })
}

describe('CheckoutPage', () => {
  it('renders the three sections and the server-computed summary', async () => {
    renderCheckout()
    await settle()

    // Headings, not bare text: the step indicator also says "Payment", and
    // the section's identity is its role, not its glyphs.
    expect(screen.getByRole('heading', { name: 'Contact' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Delivery address' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Payment' })).toBeInTheDocument()

    // The summary's figures are the CART RESPONSE's, not client arithmetic:
    // subtotal 2800 + shipping 400, with the dram total as the muted second.
    // ($28.00 appears twice: the line total and the subtotal row agree.)
    expect(screen.getAllByText('$28.00')).toHaveLength(2)
    expect(screen.getByText('$4.00')).toBeInTheDocument()
    expect(screen.getByText('$32.00')).toBeInTheDocument()
    // A PLAIN space here, though formatMoney emits a non-breaking one:
    // Testing Library normalizes the node's text (NBSP collapses to a
    // space) but compares the matcher string verbatim.
    expect(screen.getByText('15,300 ֏')).toBeInTheDocument()
  })

  it('an empty submit fails client-side with the server’s field keys — and no request', async () => {
    renderCheckout()
    await settle()
    const before = requests.length

    fireEvent.click(screen.getByRole('button', { name: 'Place the order' }))

    // One message per missing field, all at once.
    expect(await screen.findAllByRole('alert')).toHaveLength(6)
    // Nothing left the page: same requests as before the click.
    expect(requests.length).toBe(before)
    // Focus moved to the FIRST invalid input.
    expect(screen.getByLabelText('First name')).toHaveFocus()
  })

  it('a filled form posts choices and no money, then navigates to the order', async () => {
    renderCheckout()
    await settle()
    fillAddress()
    fireEvent.click(screen.getByRole('radio', { name: /Bank transfer/ }))
    fireEvent.click(screen.getByLabelText('Leave with the neighbour if I am out'))
    fireEvent.click(screen.getByRole('button', { name: 'Place the order' }))

    await screen.findByText('order page')

    const post = requests.find((r) => r.body !== undefined)
    expect(post).toBeDefined()
    expect(post!.url).toContain('/api/v1/orders')
    expect(post!.body).toEqual({
      address: {
        first_name: 'Anahit',
        last_name: 'Sargsyan',
        phone: '+374 91 000000',
        street: '14 Abovyan St, apt 6',
        city: 'Yerevan',
        postal_code: '0009',
        country: 'AM',
      },
      payment_method: 'bank_transfer',
      delivery_note: '',
      leave_with_neighbour: true,
    })
    // The security property, asserted from the client side too: the body
    // carries no total, no prices, no currency — those are the server's.
    expect(Object.keys(post!.body as object)).not.toContain('total_minor')
  })

  it('cash on delivery is disabled while shopping in dollars', async () => {
    renderCheckout()
    await settle()

    // The design's own words are the reason: "Cash — on delivery, AMD only".
    expect(screen.getByRole('radio', { name: /Cash/ })).toBeDisabled()
  })
})
