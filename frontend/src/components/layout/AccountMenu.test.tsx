import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AccountMenu } from './AccountMenu'
import type { User } from '../../api/types'

/**
 * The menu-button contract (WAI-ARIA APG): the button owns aria-haspopup /
 * aria-expanded; opening moves REAL focus into the items (unlike
 * PillSelect's combobox, which points with aria-activedescendant); arrows
 * cycle, Escape closes and hands focus back to the button. These are the
 * behaviours a keyboard or screen-reader user relies on; styling is free to
 * change.
 */

const user: User = {
  id: 1,
  email: 'anahit@example.com',
  role: 'customer',
  hive: { prior_orders: 7, member: true, member_discount_percent: 8, first_delivery_free: false },
}

beforeEach(() => {
  // useLogout only fires on the sign-out item; the stub keeps jsdom from
  // attempting a real network call if a test clicks it.
  vi.stubGlobal(
    'fetch',
    vi.fn(() => Promise.resolve(new Response(null, { status: 204 }))),
  )
})
afterEach(() => vi.unstubAllGlobals())

function renderMenu() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <AccountMenu user={user} />
      </QueryClientProvider>
    </MemoryRouter>,
  )
  return screen.getByRole('button', { name: 'anahit' })
}

describe('AccountMenu', () => {
  it('is closed by default and labelled with the email local part', () => {
    const button = renderMenu()
    expect(button).toHaveAttribute('aria-haspopup', 'menu')
    expect(button).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })

  it('opens on click with the four screens and sign-out as menu items', () => {
    const button = renderMenu()
    fireEvent.click(button)

    expect(button).toHaveAttribute('aria-expanded', 'true')
    const items = screen.getAllByRole('menuitem')
    expect(items.map((el) => el.textContent)).toEqual([
      'My orders',
      'Wishlist',
      'Addresses',
      'Settings',
      'Sign out',
    ])
    expect(screen.getByRole('menuitem', { name: 'My orders' })).toHaveAttribute(
      'href',
      '/account/orders',
    )
  })

  it('ArrowDown on the button opens and focuses the first item', async () => {
    const button = renderMenu()
    fireEvent.keyDown(button, { key: 'ArrowDown' })

    // Focus lands a frame later (the menu enters the DOM on this render).
    await waitFor(() =>
      expect(screen.getByRole('menuitem', { name: 'My orders' })).toHaveFocus(),
    )
  })

  it('arrows cycle through items and wrap around', async () => {
    const button = renderMenu()
    fireEvent.keyDown(button, { key: 'ArrowUp' }) // opens on the LAST item
    const menu = await screen.findByRole('menu')
    await waitFor(() =>
      expect(screen.getByRole('menuitem', { name: 'Sign out' })).toHaveFocus(),
    )

    fireEvent.keyDown(menu, { key: 'ArrowDown' }) // wraps to the first
    expect(screen.getByRole('menuitem', { name: 'My orders' })).toHaveFocus()
    fireEvent.keyDown(menu, { key: 'ArrowDown' })
    expect(screen.getByRole('menuitem', { name: 'Wishlist' })).toHaveFocus()
  })

  it('Escape closes and returns focus to the button', async () => {
    const button = renderMenu()
    fireEvent.click(button)
    const menu = screen.getByRole('menu')

    fireEvent.keyDown(menu, { key: 'Escape' })

    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
    expect(button).toHaveFocus()
    expect(button).toHaveAttribute('aria-expanded', 'false')
  })

  it('choosing a destination closes the menu', () => {
    const button = renderMenu()
    fireEvent.click(button)
    fireEvent.click(screen.getByRole('menuitem', { name: 'Wishlist' }))
    expect(screen.queryByRole('menu')).not.toBeInTheDocument()
  })
})
