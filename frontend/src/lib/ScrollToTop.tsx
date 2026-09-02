import { useEffect, useRef } from 'react'
import { useLocation, useNavigationType } from 'react-router'
import { DEFAULT_LOCALE, pathForLocale } from '../i18n/locales'

/**
 * Scroll to the top when the visitor moves to a DIFFERENT page.
 *
 * A browser only resets scroll on a real page load. An SPA never loads a
 * page again after the first one — clicking a product card swaps components
 * inside the same document — so a visitor who was 2000px down the home page
 * opened the product page 2000px down too. (React Router's own
 * <ScrollRestoration> solves this, but only for data routers; this app uses
 * the declarative <BrowserRouter>.)
 *
 * Three navigations deliberately do NOT scroll:
 *
 *  - Back/Forward (`POP`): the browser restores its own remembered position,
 *    and "take me back to where I was" is the whole point of the button.
 *  - A language switch: /products/honey → /hy/products/honey is a new URL
 *    but the SAME page, so the reader keeps their place in the text. The
 *    page's identity is the locale-STRIPPED path, which is also why plain
 *    `pathname` is the wrong key here.
 *  - A hash link: the browser is about to jump to the anchor; scrolling to
 *    the top first would fight it.
 *
 * Query-string changes (shop filters, pagination) keep the pathname, so
 * they never scroll — filtering while staying put is the behaviour the
 * filter sidebar already relies on. But the FOOTER's category links produce
 * the very same transition (/shop → /shop?category=honey) wanting the very
 * opposite: the reader is at the bottom of the page and asked for a fresh
 * one. Same URL change, two intents — so the intent travels WITH the
 * navigation: a link that passes `state={scrollToTopState}` scrolls
 * unconditionally, same page or not.
 */

export function ScrollToTop() {
  const location = useLocation()
  const navigationType = useNavigationType()

  const pageKey = pathForLocale(location.pathname, DEFAULT_LOCALE)
  // Seeded with the FIRST page's key, so the initial render never scrolls —
  // on a fresh load or reload the browser's own restoration is in charge.
  const previous = useRef(pageKey)

  // Depending on `location` (a new object per navigation), not just the
  // derived key: a footer link clicked ON its own page changes neither
  // pathname nor search, and the effect still has to run to see its state.
  useEffect(() => {
    const changed = previous.current !== pageKey
    previous.current = pageKey

    if (navigationType === 'POP') return
    const state = location.state as { scrollToTop?: boolean } | null
    if (state?.scrollToTop) {
      window.scrollTo(0, 0)
      return
    }
    if (!changed || location.hash) return
    window.scrollTo(0, 0)
  }, [location, navigationType, pageKey])

  return null
}
