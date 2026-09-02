import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useHoverSlideshow } from './useHoverSlideshow'

/**
 * The slideshow's contract: advance on a timer while hovered, snap back to
 * the hero on leave — and stay INERT for a single photo, under
 * prefers-reduced-motion, and on devices whose pointer cannot hover (where
 * a tap fires mouseenter with no cursor to ever leave).
 *
 * jsdom implements no matchMedia at all, so each test states the device it
 * is pretending to be. That default also means every OTHER component test
 * in the suite runs with the slideshow off — which is the correct rendering
 * for an environment that cannot hover.
 */

function pretendDevice({ reducedMotion = false, canHover = true } = {}) {
  vi.stubGlobal(
    'matchMedia',
    vi.fn((query: string) => ({
      matches: query.includes('prefers-reduced-motion') ? reducedMotion : canHover,
    })),
  )
}

beforeEach(() => vi.useFakeTimers())
afterEach(() => {
  vi.useRealTimers()
  vi.unstubAllGlobals()
})

describe('useHoverSlideshow', () => {
  it('advances every interval while hovered, wrapping past the end', () => {
    pretendDevice()
    const { result } = renderHook(() => useHoverSlideshow(3, 900))

    act(() => result.current.onMouseEnter())
    expect(result.current.cycling).toBe(true)
    expect(result.current.index).toBe(0)

    act(() => vi.advanceTimersByTime(900))
    expect(result.current.index).toBe(1)
    act(() => vi.advanceTimersByTime(900))
    expect(result.current.index).toBe(2)
    // ...and around: modulo, not clamp — the cycle should loop, not stall
    // on the last photo.
    act(() => vi.advanceTimersByTime(900))
    expect(result.current.index).toBe(0)
  })

  it('resets to the hero on leave, but stays warm', () => {
    pretendDevice()
    const { result } = renderHook(() => useHoverSlideshow(3))

    act(() => result.current.onMouseEnter())
    act(() => vi.advanceTimersByTime(900))
    expect(result.current.index).toBe(1)

    act(() => result.current.onMouseLeave())
    expect(result.current.index).toBe(0)
    expect(result.current.cycling).toBe(false)
    // warm is sticky: the non-hero images are already fetched, so unmounting
    // them on leave would only re-cost the requests on the next hover.
    expect(result.current.warm).toBe(true)

    // The timer really stopped — time passing at rest must not move it.
    act(() => vi.advanceTimersByTime(5000))
    expect(result.current.index).toBe(0)
  })

  it('is inert for fewer than two photos', () => {
    pretendDevice()
    const { result } = renderHook(() => useHoverSlideshow(1))

    act(() => result.current.onMouseEnter())
    expect(result.current.cycling).toBe(false)
    expect(result.current.warm).toBe(false)
  })

  it('is inert under prefers-reduced-motion', () => {
    pretendDevice({ reducedMotion: true })
    const { result } = renderHook(() => useHoverSlideshow(3))

    act(() => result.current.onMouseEnter())
    expect(result.current.cycling).toBe(false)
    act(() => vi.advanceTimersByTime(3000))
    expect(result.current.index).toBe(0)
  })

  it('is inert when the pointer cannot hover (touch)', () => {
    pretendDevice({ canHover: false })
    const { result } = renderHook(() => useHoverSlideshow(3))

    act(() => result.current.onMouseEnter())
    expect(result.current.cycling).toBe(false)
  })
})
