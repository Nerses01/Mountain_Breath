import { afterEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { VideoEditor } from './ProductContentEditor'
import type { AdminProduct, ProductDetail, ProductVideo } from '../../api/types'

/**
 * The video slot's contract: empty shows an upload control, filled shows a
 * preview and a delete that goes through the SHARED images DELETE route —
 * the video is a gallery row, so no video-specific delete endpoint exists
 * to drift from the photos'.
 */

const product: AdminProduct = {
  id: 5,
  category_id: 1,
  category_slug: 'royal-jelly',
  category_name: 'Royal jelly',
  rating_avg: 0,
  rating_count: 0,
  slug: 'fresh-royal-jelly',
  name: 'Fresh Royal Jelly',
  description: '',
  images: [],
  created_at: '2026-08-01T00:00:00Z',
  variants: [],
  badge: '',
  badge_tone: 'honey',
  benefits: [],
  currency: 'USD',
  is_active: true,
}

function detailWith(video: ProductVideo | null): ProductDetail {
  return {
    ...product,
    images: [],
    video,
    highlights: [],
    usage_cards: [],
    disclaimer: '',
    storage_note: '',
    harvest_note: '',
    shipping_note: '',
    lab_batch: '',
    is_cold_chain: true,
    can_review: false,
  }
}

let fetchMock: ReturnType<typeof vi.fn>

function stubFetch(detail: ProductDetail) {
  fetchMock = vi.fn((_input: RequestInfo | URL, init?: RequestInit) => {
    // The DELETE answers 204; everything else (the detail read) answers the
    // product. The component never fetches anything besides these two.
    if (init?.method === 'DELETE') {
      return Promise.resolve(new Response(null, { status: 204 }))
    }
    return Promise.resolve(
      new Response(JSON.stringify(detail), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
  vi.stubGlobal('fetch', fetchMock)
}

afterEach(() => vi.unstubAllGlobals())

function renderEditor() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <MemoryRouter>
      <QueryClientProvider client={qc}>
        <VideoEditor product={product} />
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

describe('VideoEditor', () => {
  it('offers an upload when the slot is empty', async () => {
    stubFetch(detailWith(null))
    renderEditor()

    expect(
      await screen.findByRole('button', { name: /upload video/i }),
    ).toBeInTheDocument()
    // The picker is limited to what the server's sniffer will accept — the
    // real gate is the magic-number check, this only saves a round trip.
    expect(screen.getByTestId('video-file-input')).toHaveAttribute(
      'accept',
      'video/mp4,video/webm',
    )
  })

  it('shows the clip and deletes it through the shared images route', async () => {
    stubFetch(detailWith({ id: 42, url: '/uploads/p5-clip.mp4', alt: '' }))
    renderEditor()

    const del = await screen.findByRole('button', { name: 'delete video' })
    expect(screen.queryByRole('button', { name: /upload video/i })).not.toBeInTheDocument()

    fireEvent.click(del)

    await waitFor(() => {
      const deleteCall = fetchMock.mock.calls.find(
        ([, init]) => (init as RequestInit | undefined)?.method === 'DELETE',
      )
      expect(String(deleteCall?.[0])).toContain('/api/v1/admin/products/5/images/42')
    })
  })
})
