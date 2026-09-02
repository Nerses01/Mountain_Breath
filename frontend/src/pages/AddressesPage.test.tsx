import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { AddressesPage } from './AddressesPage'

/**
 * A4's new behaviours: the card grid renders the book with the neighbour
 * note row, and Remove is a TWO-step control — the first press arms it
 * ("Remove?"), only the second fires the DELETE. The CRUD itself is E8's
 * and stays covered by the store suite.
 */

const entries = [
  {
    id: 1, label: 'Home', is_default: true, leave_with_neighbour: true,
    first_name: 'Anahit', last_name: 'Sargsyan', phone: '+374 91 000000',
    street: '14 Abovyan St', city: 'Yerevan', postal_code: '0009', country: 'AM',
  },
  {
    id: 2, label: 'Office', is_default: false, leave_with_neighbour: false,
    first_name: 'Anahit', last_name: 'Sargsyan', phone: '+374 91 000000',
    street: '2 Vazgen Sargsyan St', city: 'Yerevan', postal_code: '0010', country: 'AM',
  },
]

let deletes: string[] = []

beforeEach(() => {
  deletes = []
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (init?.method === 'DELETE') {
        deletes.push(url)
        return Promise.resolve(new Response(null, { status: 204 }))
      }
      return Promise.resolve(
        new Response(JSON.stringify(url.includes('/addresses') ? entries : {}), {
          status: 200,
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
        <AddressesPage />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('AddressesPage', () => {
  it('renders the book as cards with the Default badge', async () => {
    renderPage()
    expect(await screen.findByText('Home')).toBeInTheDocument()
    expect(screen.getByText('Office')).toBeInTheDocument()
    expect(screen.getByText('Default')).toBeInTheDocument()
    // Both cards carry the neighbour note row, whatever its state.
    expect(screen.getAllByText('Leave with the neighbour if I am out')).toHaveLength(2)
  })

  it('Remove arms on the first press and deletes only on the second', async () => {
    renderPage()
    await screen.findByText('Home')

    const [removeButton] = screen.getAllByRole('button', { name: 'Delete' })
    fireEvent.click(removeButton)

    // Armed, not fired: the DELETE has not gone out.
    expect(deletes).toHaveLength(0)
    expect(removeButton).toHaveTextContent('Remove?')

    fireEvent.click(removeButton)
    // The mutation runs on a microtask — the DELETE arrives, not instantly.
    await waitFor(() => expect(deletes).toHaveLength(1))
    expect(deletes[0]).toContain('/account/addresses/1')
  })

  it('the arming lapses after three seconds without a second press', async () => {
    vi.useFakeTimers()
    try {
      renderPage()
      // findBy* uses real timers internally; flush the query manually.
      await vi.waitFor(() => {
        if (!screen.queryByText('Home')) throw new Error('not yet')
      })

      const [removeButton] = screen.getAllByRole('button', { name: 'Delete' })
      fireEvent.click(removeButton)
      expect(removeButton).toHaveTextContent('Remove?')

      // The timeout's setState is a React update — advancing the clock
      // must happen inside act() or the re-render is left pending.
      act(() => vi.advanceTimersByTime(3100))
      expect(removeButton).toHaveTextContent('Delete')
      expect(deletes).toHaveLength(0)
    } finally {
      vi.useRealTimers()
    }
  })
})
