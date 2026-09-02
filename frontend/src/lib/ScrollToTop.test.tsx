import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { MemoryRouter, useNavigate } from 'react-router'
import { ScrollToTop } from './ScrollToTop'
import { scrollToTopState } from './scrollToTopState'

/**
 * The contract: moving to a DIFFERENT page scrolls to the top; a language
 * switch (same page, other prefix) and Back/Forward do not. jsdom has no
 * layout, so `window.scrollTo` is a stub we watch rather than a scrollbar
 * we could read.
 */

// Buttons that drive the router the way the app does — Link clicks are
// PUSHes, the browser's Back button is a POP (navigate(-1)).
function Probe() {
  const navigate = useNavigate()
  return (
    <>
      <button onClick={() => navigate('/products/honey')}>to product</button>
      <button onClick={() => navigate('/hy/products/honey')}>to armenian</button>
      <button onClick={() => navigate(-1)}>back</button>
      <button onClick={() => navigate('/shop?category=honey')}>filter</button>
      <button onClick={() => navigate('/shop?category=honey', { state: scrollToTopState })}>
        footer category
      </button>
    </>
  )
}

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <ScrollToTop />
      <Probe />
    </MemoryRouter>,
  )
}

describe('ScrollToTop', () => {
  const scrollTo = vi.fn()

  beforeEach(() => {
    scrollTo.mockClear()
    window.scrollTo = scrollTo as unknown as typeof window.scrollTo
  })

  it('scrolls to the top when a new page opens', () => {
    renderAt('/')

    fireEvent.click(screen.getByText('to product'))

    expect(scrollTo).toHaveBeenCalledWith(0, 0)
  })

  it('never scrolls on the first render', () => {
    renderAt('/products/honey')

    expect(scrollTo).not.toHaveBeenCalled()
  })

  it('stays put on a language switch — same page, other prefix', () => {
    renderAt('/products/honey')

    fireEvent.click(screen.getByText('to armenian'))

    expect(scrollTo).not.toHaveBeenCalled()
  })

  it('leaves Back alone so the browser can restore its position', () => {
    renderAt('/')
    fireEvent.click(screen.getByText('to product'))
    scrollTo.mockClear()

    fireEvent.click(screen.getByText('back'))

    expect(scrollTo).not.toHaveBeenCalled()
  })

  it('stays put on a query-only change — the sidebar filters in place', () => {
    renderAt('/shop')

    fireEvent.click(screen.getByText('filter'))

    expect(scrollTo).not.toHaveBeenCalled()
  })

  it('scrolls on the same query-only change when the link requests it', () => {
    // The footer's category links: same /shop → /shop?category transition
    // as the sidebar, opposite intent — carried as navigation state.
    renderAt('/shop')

    fireEvent.click(screen.getByText('footer category'))

    expect(scrollTo).toHaveBeenCalledWith(0, 0)
  })
})
