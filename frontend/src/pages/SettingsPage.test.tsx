import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { CurrencyProvider } from '../lib/CurrencyProvider'
import { SettingsPage } from './SettingsPage'

/**
 * A5's screen contracts: the password form posts both fields to the right
 * endpoint, the order-updates switch PATCHes through, and the two
 * sender-less channels render as DISABLED switches (decision #87) — the
 * screen's honesty is a testable property.
 */

let requests: { url: string; method: string; body: unknown }[] = []

beforeEach(() => {
  requests = []
  localStorage.clear()
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      requests.push({
        url,
        method: init?.method ?? 'GET',
        body: init?.body ? JSON.parse(String(init.body)) : undefined,
      })
      let payload: unknown = {}
      let status = 200
      if (url.includes('/auth/me')) {
        payload = {
          id: 1, email: 'anahit@example.com', role: 'customer',
          full_name: 'Anahit Sargsyan', phone: '',
          hive: { prior_orders: 2, member: true, member_discount_percent: 8, first_delivery_free: false },
        }
      } else if (url.includes('/account/notifications')) {
        if (init?.method === 'PATCH') status = 204
        else payload = { order_updates: true, newsletter: 'none' }
      } else if (url.includes('/account/password')) {
        status = 204
      }
      return Promise.resolve(
        new Response(status === 204 ? null : JSON.stringify(payload), {
          status,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    }),
  )
})
afterEach(() => vi.unstubAllGlobals())

function renderPage() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <CurrencyProvider>
          <SettingsPage />
        </CurrencyProvider>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('SettingsPage', () => {
  it('shows the profile with the not-set fallback for the empty phone', async () => {
    renderPage()
    expect(await screen.findByText('Anahit Sargsyan')).toBeInTheDocument()
    expect(screen.getByText('Not set yet')).toBeInTheDocument()
  })

  it('the password form posts both fields and reports the sign-out promise', async () => {
    renderPage()
    fireEvent.click(await screen.findByRole('button', { name: 'Change' }))

    fireEvent.change(screen.getByLabelText('Current password'), {
      target: { value: 'old-pass-123' },
    })
    fireEvent.change(screen.getByLabelText('New password'), {
      target: { value: 'new-pass-456' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Change password' }))

    await screen.findByText('Password changed — your other devices were signed out.')
    const post = requests.find((r) => r.url.includes('/account/password'))
    expect(post?.method).toBe('POST')
    expect(post?.body).toEqual({
      current_password: 'old-pass-123',
      new_password: 'new-pass-456',
    })
  })

  it('the order-updates switch PATCHes the preference', async () => {
    renderPage()
    const toggle = await screen.findByRole('switch', { name: 'Order updates' })
    await waitFor(() => expect(toggle).toBeEnabled())

    fireEvent.click(toggle)
    await waitFor(() => {
      const patch = requests.find(
        (r) => r.url.includes('/account/notifications') && r.method === 'PATCH',
      )
      expect(patch?.body).toEqual({ order_updates: false })
    })
  })

  // Since decision #90 these segments are the shop's ONLY currency
  // control — the footer switcher is gone — so their behaviour is pinned
  // here instead of in a switcher test.
  it('the currency segments set the market and the display mode', async () => {
    renderPage()
    const amd = await screen.findByRole('button', { name: 'AMD' })
    expect(screen.getByRole('button', { name: 'USD + AMD' })).toHaveAttribute(
      'aria-pressed',
      'true',
    )

    fireEvent.click(amd)

    expect(amd).toHaveAttribute('aria-pressed', 'true')
    expect(localStorage.getItem('mb_currency')).toBe('AMD')
    expect(localStorage.getItem('mb_currency_display')).toBe('single')
  })

  it('the sender-less channels are switches you cannot flip', async () => {
    renderPage()
    expect(await screen.findByRole('switch', { name: 'Wishlist alerts' })).toBeDisabled()
    expect(screen.getByRole('switch', { name: 'SMS on delivery day' })).toBeDisabled()
    // The delete row's button is the F2 stub — visible, inert.
    expect(screen.getByRole('button', { name: 'Delete…' })).toBeDisabled()
  })
})
