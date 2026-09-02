import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { HomePage } from './HomePage'

/**
 * The hero image's contract (F3, first real image in the shop): it is a real
 * <img> with a translated alt — a screen reader hears what the shop sells,
 * not silence — and it declares intrinsic width/height so the browser
 * reserves the box before the file arrives (no layout shift under the
 * headline). The test pins the CONTRACT, not the pixels: jsdom never decodes
 * the jpg, which is exactly why the attributes have to be asserted — nothing
 * else in the suite would notice them disappearing.
 */

beforeEach(() => {
  // The page also fires the six-card shelf query; an empty page keeps the
  // test about the hero, not the shelf.
  vi.stubGlobal(
    'fetch',
    vi.fn(() =>
      Promise.resolve(
        new Response(JSON.stringify({ items: [] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    ),
  )
})
afterEach(() => vi.unstubAllGlobals())

function renderHome() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter initialEntries={['/']}>
      <QueryClientProvider client={qc}>
        <HomePage />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('HomePage hero', () => {
  it('renders the photo with a translated alt and explicit dimensions', () => {
    renderHome()
    const img = screen.getByRole('img', {
      name: /glass jar of raw honey/i,
    })
    expect(img).toHaveAttribute('src')
    expect(img).toHaveAttribute('width', '1024')
    expect(img).toHaveAttribute('height', '1024')
  })
})
