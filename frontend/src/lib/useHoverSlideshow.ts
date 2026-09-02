import { useEffect, useRef, useState } from 'react'

/**
 * The card's hover slideshow (decision #99): while the cursor rests on a
 * product widget, its photos advance on a timer; leaving snaps back to the
 * hero. Extracted for the same reason as useAddToCartFlash — ProductCard and
 * WishlistCard both want it, and the second hand-written copy is where
 * behaviour starts to drift.
 *
 * The hook is the state machine; rendering (stacked images, dots) stays with
 * the caller. It is deliberately INERT — handlers that do nothing — when:
 *
 *  - there is nothing to cycle (fewer than two photos);
 *  - the reader asked for reduced motion. An unprompted animation under the
 *    cursor is exactly what that setting refuses;
 *  - the pointer cannot hover. On touch screens a TAP fires mouseenter, and
 *    without this gate it would start a timer with no cursor to ever leave.
 *
 * The capability checks run at hover time, not at mount: matchMedia answers
 * the CURRENT state, so a toggled OS setting is honoured on the next hover
 * with no listeners to manage.
 *
 * `warm` is the lazy-loading half of the contract: it flips true on the
 * first hover and never back. Before it, a card should mount only its hero
 * <img>; after it, mounting all of them lets the browser fetch the rest
 * exactly when they are about to be shown — no blank frames, no manual
 * preloading, and a grid of twelve cards costs twelve images until someone
 * actually hovers one.
 */
export function useHoverSlideshow(count: number, intervalMs = 900) {
  const [index, setIndex] = useState(0)
  const [cycling, setCycling] = useState(false)
  const [warm, setWarm] = useState(false)
  const timer = useRef<ReturnType<typeof setInterval>>(undefined)

  // The timer dies with the card — no setState on an unmounted component.
  useEffect(() => () => clearInterval(timer.current), [])

  const canCycle = () =>
    count > 1 &&
    typeof window.matchMedia === 'function' &&
    !window.matchMedia('(prefers-reduced-motion: reduce)').matches &&
    window.matchMedia('(hover: hover) and (pointer: fine)').matches

  const onMouseEnter = () => {
    if (timer.current !== undefined || !canCycle()) return
    setWarm(true)
    setCycling(true)
    timer.current = setInterval(() => setIndex((i) => (i + 1) % count), intervalMs)
  }

  const onMouseLeave = () => {
    if (timer.current === undefined) return
    clearInterval(timer.current)
    timer.current = undefined
    setCycling(false)
    // Back to the hero: the card at rest always shows the photo the shop
    // chose, so a grid never ends up a patchwork of whatever frame each
    // cursor happened to leave behind.
    setIndex(0)
  }

  return { index, cycling, warm, onMouseEnter, onMouseLeave }
}
