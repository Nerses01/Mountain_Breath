/**
 * Pass as a react-router Link/navigate `state` to make ScrollToTop scroll
 * even when the URL barely changes (a query-only filter link in the footer,
 * or a link to the page it is already on). Lives in its own module rather
 * than ScrollToTop.tsx because a component file that also exports constants
 * loses Fast Refresh.
 */
export const scrollToTopState = { scrollToTop: true } as const
