import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { NewsletterConfirmPage } from './NewsletterPages'

/**
 * The consent page's contract: NOTHING fires on load — the button press is
 * the consent, because link scanners follow (and some execute) whatever a
 * page does automatically. Then the token from the URL is what travels.
 */

let requests: { url: string; body: unknown }[] = []

beforeEach(() => {
  requests = []
  vi.stubGlobal(
    'fetch',
    vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      requests.push({
        url: String(input),
        body: init?.body ? JSON.parse(String(init.body)) : undefined,
      })
      return Promise.resolve(new Response(null, { status: 204 }))
    }),
  )
})
afterEach(() => vi.unstubAllGlobals())

function renderConfirm() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter initialEntries={['/newsletter/confirm/raw-token']}>
      <QueryClientProvider client={qc}>
        <Routes>
          <Route path="/newsletter/confirm/:token" element={<NewsletterConfirmPage />} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('NewsletterConfirmPage', () => {
  it('fires nothing on load — the button press is the consent', async () => {
    renderConfirm()
    expect(screen.getByRole('button', { name: 'Confirm my subscription' })).toBeInTheDocument()
    expect(requests).toHaveLength(0)

    fireEvent.click(screen.getByRole('button', { name: 'Confirm my subscription' }))
    expect(await screen.findByRole('status')).toHaveTextContent(/on the list/)
    expect(requests).toHaveLength(1)
    expect(requests[0].url).toContain('/newsletter/confirm')
    expect(requests[0].body).toEqual({ token: 'raw-token' })
  })
})
