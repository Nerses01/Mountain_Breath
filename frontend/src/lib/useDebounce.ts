import { useEffect, useState } from 'react'

// Returns `value`, but only after it has stopped changing for `delayMs`.
// Typing "honey" fires 5 state updates; the debounced value updates once,
// 300ms after the last keystroke — so we search once, not 5 times.
export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), delayMs)
    // Cleanup runs on every value change BEFORE the next effect: the
    // previous timer is cancelled — that's the entire debounce mechanism.
    return () => clearTimeout(timer)
  }, [value, delayMs])

  return debounced
}
