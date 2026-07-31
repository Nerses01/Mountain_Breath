import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import { useDebouncedValue } from './useDebounce'

describe('useDebouncedValue', () => {
  beforeEach(() => vi.useFakeTimers())
  afterEach(() => vi.useRealTimers())

  it('only exposes the value after the delay of silence', () => {
    const { result, rerender } = renderHook(({ v }) => useDebouncedValue(v, 300), {
      initialProps: { v: 'h' },
    })

    // rapid typing: each change resets the timer
    rerender({ v: 'ho' })
    rerender({ v: 'hon' })
    rerender({ v: 'honey' })
    expect(result.current).toBe('h') // still the initial value

    act(() => {
      vi.advanceTimersByTime(299)
    })
    expect(result.current).toBe('h') // not yet

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(result.current).toBe('honey') // exactly one update, the final value
  })
})
