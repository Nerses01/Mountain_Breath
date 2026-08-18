import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AccountLayout } from './AccountLayout'
import { LegacyRedirect } from '../../App'

/**
 * A1's contracts at the component boundary:
 *
 *  - the shell guards ONCE: signed out renders the sign-in prompt and no
 *    rail, signed in renders the rail and the child pane;
 *  - the rail's nav marks the current screen with aria-current="page",
 *    including on a CHILD of the section (/account/orders/42 lights
 *    "My orders" — the reason it uses NavLink's prefix match);
 *  - the legacy paths redirect into the shell, keeping the locale prefix
 *    and any :id param — emailed links must keep working.
 */

const user = {
  id: 1,
  email: 'anahit@example.com',
  role: 'customer',
  hive: { prior_orders: 7, member: true, member_discount_percent: 8, first_delivery_free: false },
}

// URL-keyed stub: one fetch serves every query the shell fires.
function stubFetch(me: unknown | null) {
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL) => {
      const url = String(input)
      let status = 200
      let payload: unknown = []
      if (url.includes('/auth/me')) {
        if (me === null) status = 401
        payload = me ?? { error: { code: 'unauthorized', message: '' } }
      } else if (url.includes('/orders')) {
        payload = [{ id: 1 }, { id: 2 }]
      } else if (url.includes('/wishlist')) {
        payload = [{ id: 10 }, { id: 11 }, { id: 12 }]
      } else if (url.includes('/addresses')) {
        payload = []
      }
      return Promise.resolve(
        new Response(JSON.stringify(payload), {
          status,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    }),
  )
}

beforeEach(() => stubFetch(user))
afterEach(() => vi.unstubAllGlobals())

function renderAt(path: string) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={qc}>
        <Routes>
          <Route path="/account" element={<AccountLayout />}>
            <Route path="orders" element={<p>orders pane</p>} />
            <Route path="orders/:id" element={<p>order detail pane</p>} />
            <Route path="wishlist" element={<p>wishlist pane</p>} />
          </Route>
          <Route path="/orders/:id" element={<LegacyRedirect to="/account/orders/:id" />} />
          <Route path="/wishlist" element={<LegacyRedirect to="/account/wishlist" />} />
          <Route path="/hy/wishlist" element={<LegacyRedirect to="/account/wishlist" />} />
          <Route path="/hy/account/wishlist" element={<p>hy wishlist pane</p>} />
          <Route path="/login" element={<p>login page</p>} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('AccountLayout', () => {
  it('signed out: renders the one sign-in prompt and no rail', async () => {
    stubFetch(null)
    renderAt('/account/orders')

    expect(await screen.findByRole('link', { name: 'sign in' })).toBeInTheDocument()
    expect(screen.queryByText('My orders')).not.toBeInTheDocument()
    expect(screen.queryByText('orders pane')).not.toBeInTheDocument()
  })

  it('signed in: rail with counts around the child pane', async () => {
    renderAt('/account/orders')

    expect(await screen.findByText('orders pane')).toBeInTheDocument()
    // The stub's two orders and three wishlist rows become the nav counts.
    // waitFor, not a plain expect: the pane renders as soon as /auth/me
    // resolves, while the count queries are still in flight — a count is a
    // thing that ARRIVES, so the assertion must be allowed to wait for it.
    await waitFor(() =>
      expect(screen.getByRole('link', { name: /My orders/ })).toHaveTextContent('2'),
    )
    expect(screen.getByRole('link', { name: /Wishlist/ })).toHaveTextContent('3')
    // The profile card shows the identity we have pre-A5: the email.
    expect(screen.getByText('anahit@example.com')).toBeInTheDocument()
    expect(screen.getByText('Hive club member')).toBeInTheDocument()
  })

  it('marks the active section with aria-current, even on a child route', async () => {
    renderAt('/account/orders/42')

    expect(await screen.findByText('order detail pane')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /My orders/ })).toHaveAttribute(
      'aria-current',
      'page',
    )
    expect(screen.getByRole('link', { name: /Wishlist/ })).not.toHaveAttribute('aria-current')
  })

  it('legacy /orders/:id redirects into the shell with the id intact', async () => {
    renderAt('/orders/42')
    expect(await screen.findByText('order detail pane')).toBeInTheDocument()
  })

  it('legacy redirect keeps the locale prefix', async () => {
    renderAt('/hy/wishlist')
    expect(await screen.findByText('hy wishlist pane')).toBeInTheDocument()
  })
})
