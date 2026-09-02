import { useEffect, useRef, useState } from 'react'
import type { Product } from '../api/types'

/**
 * The "In cart: N" confirmation flash behind every Add button — extracted
 * when the wishlist's cards wanted the same behaviour the shop's cards had,
 * because the second copy is where the pattern starts to drift (see
 * useQuickAdd's history: three hand-written add handlers, two of them
 * stale).
 *
 * The rules it owns:
 *  - feedback shows only for a count the SERVER confirmed (the resolved
 *    number of a Promise-returning handler, or an explicit flash(n));
 *  - a second click restarts the window rather than being cut short by the
 *    first click's timer;
 *  - the timer dies with the component — no setState on an unmounted card.
 *
 * Rendering stays with the caller (label, pop animation, live region):
 * this hook is the state machine, not the pixels.
 */
export function useAddToCartFlash(onAdd?: (product: Product) => void | Promise<number>) {
  const [addedQty, setAddedQty] = useState<number | null>(null)
  const timer = useRef<ReturnType<typeof setTimeout>>(undefined)
  useEffect(() => () => clearTimeout(timer.current), [])

  const flash = (count: number) => {
    setAddedQty(count)
    clearTimeout(timer.current)
    timer.current = setTimeout(() => setAddedQty(null), 1800)
  }

  const handleAdd = async (product: Product) => {
    if (!onAdd) return
    try {
      const count = await onAdd(product)
      if (typeof count === 'number' && count > 0) flash(count)
    } catch {
      // A failed add keeps the resting label; the cart cache stays truthful.
    }
  }

  return { addedQty, handleAdd, flash }
}
