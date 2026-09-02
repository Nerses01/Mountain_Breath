import { useEffect } from 'react'
import { useLocation } from 'react-router'
import { LOCALES, localeFromPath, pathForLocale } from '../i18n/locales'

/**
 * Per-page <head> management (E10) — titles, description, OG tags, the
 * canonical URL and hreflang alternates. Hand-rolled rather than a helmet
 * dependency: the whole job is setting a handful of DOM nodes in an
 * effect, and the libraries' extra weight buys SSR coordination this SPA
 * does not do.
 *
 * A NOTE ON HONESTY: meta managed from JavaScript reaches crawlers that
 * execute it (Google does; some do not). The real fix is prerendering or
 * SSR — parked in Phase 11 with hosting-dependent work. What this hook
 * guarantees today: correct titles in tabs/history/bookmarks for humans,
 * and correct signals for the crawlers that matter most.
 */
export function usePageMeta({
  title,
  description,
  canonicalPath,
  jsonLd,
}: {
  /** Page name; the brand is appended. Empty = the brand alone. */
  title?: string
  description?: string
  /**
   * Locale-less path for canonical + hreflang (e.g. "/shop", stripped of
   * filters — a thousand filter permutations must not compete with the
   * clean page in an index). Defaults to the current path without query.
   */
  canonicalPath?: string
  /** Structured data (schema.org), serialized into one script tag. */
  jsonLd?: object
}) {
  const { pathname } = useLocation()

  useEffect(() => {
    document.title = title ? `${title} — Mountain Breath` : 'Mountain Breath'

    const path = canonicalPath ?? stripLocale(pathname)
    const origin = window.location.origin
    const locale = localeFromPath(pathname)

    setMeta('description', description)
    setMeta('og:title', document.title, 'property')
    setMeta('og:description', description, 'property')
    setMeta('og:type', jsonLd ? 'product' : 'website', 'property')
    setMeta('og:url', origin + pathForLocale(path, locale), 'property')

    // rel=canonical names THIS page's clean URL in its own locale;
    // hreflang names every language's twin plus x-default (the bare
    // English URL, this site's stated default since E1.5).
    setLink('canonical', origin + pathForLocale(path, locale))
    for (const l of LOCALES) {
      setLink('alternate', origin + pathForLocale(path, l), l)
    }
    setLink('alternate', origin + path, 'x-default')

    setJsonLd(jsonLd)
  }, [title, description, canonicalPath, jsonLd, pathname])
}

function stripLocale(pathname: string): string {
  return pathForLocale(pathname, 'en')
}

function setMeta(name: string, content: string | undefined, attr: 'name' | 'property' = 'name') {
  let el = document.head.querySelector<HTMLMetaElement>(`meta[${attr}="${name}"]`)
  if (!content) {
    el?.remove()
    return
  }
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute(attr, name)
    document.head.appendChild(el)
  }
  el.content = content
}

function setLink(rel: string, href: string, hreflang?: string) {
  const selector = hreflang
    ? `link[rel="${rel}"][hreflang="${hreflang}"]`
    : `link[rel="${rel}"]`
  let el = document.head.querySelector<HTMLLinkElement>(selector)
  if (!el) {
    el = document.createElement('link')
    el.rel = rel
    if (hreflang) el.hreflang = hreflang
    document.head.appendChild(el)
  }
  el.href = href
}

function setJsonLd(data: object | undefined) {
  const id = 'page-jsonld'
  let el = document.getElementById(id)
  if (!data) {
    el?.remove()
    return
  }
  if (!el) {
    el = document.createElement('script')
    el.id = id
    el.setAttribute('type', 'application/ld+json')
    document.head.appendChild(el)
  }
  el.textContent = JSON.stringify(data)
}
