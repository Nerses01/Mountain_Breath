import type { Locale } from '../i18n/locales'
import { DEFAULT_LOCALE } from '../i18n/locales'

/**
 * The content pipeline (E9, decision #3): markdown files in this directory,
 * one per page per locale, BUNDLED at build time via Vite's glob import —
 * no CMS, no runtime fetch, versioned with the code. Editing a page is a
 * commit, exactly like editing the plan.
 *
 * `eager: true` + `?raw` means every file's text is IN the bundle keyed by
 * path, and a missing translation is a missing key — which the resolvers
 * below turn into the same per-locale English fallback the API applies to
 * database content. One fallback philosophy, two storage systems.
 */

const pageFiles = import.meta.glob('./pages/*.md', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const journalFiles = import.meta.glob('./journal/*.md', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

/** The static pages the header and footer link to. A union, so a typo in a
 *  route is a compile error, not a blank page. */
export type PageSlug =
  | 'our-hive'
  | 'benefits'
  | 'shipping'
  | 'contact'
  | 'terms'
  | 'privacy'

export function pageMarkdown(slug: PageSlug, locale: Locale): string | undefined {
  return (
    pageFiles[`./pages/${slug}.${locale}.md`] ??
    pageFiles[`./pages/${slug}.${DEFAULT_LOCALE}.md`]
  )
}

export interface JournalPost {
  slug: string
  title: string
  /** ISO date from the frontmatter; formatted per-locale at render. */
  date: string
  teaser: string
  body: string
}

/**
 * Frontmatter, parsed by hand: three known string fields between two `---`
 * fences is not a YAML document, and a YAML dependency for it would be the
 * parser trap in the other direction. The grammar this accepts is exactly
 * `key: value` lines — the moment a post needs more, revisit.
 */
function parseFrontmatter(raw: string): { meta: Record<string, string>; body: string } {
  const match = /^---\r?\n([\s\S]*?)\r?\n---\r?\n?/.exec(raw)
  if (!match) return { meta: {}, body: raw }

  const meta: Record<string, string> = {}
  for (const line of match[1].split(/\r?\n/)) {
    const idx = line.indexOf(':')
    if (idx > 0) {
      meta[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
    }
  }
  return { meta, body: raw.slice(match[0].length) }
}

function postFrom(slug: string, raw: string): JournalPost {
  const { meta, body } = parseFrontmatter(raw)
  return {
    slug,
    title: meta.title ?? slug,
    date: meta.date ?? '',
    teaser: meta.teaser ?? '',
    body,
  }
}

/** Every post in the given locale (English fallback per post), newest first. */
export function journalPosts(locale: Locale): JournalPost[] {
  // Slugs come from the English files — English is the reference locale for
  // content exactly as it is for UI strings: a post exists once it exists
  // in English, and translations attach to it.
  const slugs = Object.keys(journalFiles)
    .filter((path) => path.endsWith(`.${DEFAULT_LOCALE}.md`))
    .map((path) => path.slice('./journal/'.length, -`.${DEFAULT_LOCALE}.md`.length))

  return slugs
    .map((slug) => {
      const raw =
        journalFiles[`./journal/${slug}.${locale}.md`] ??
        journalFiles[`./journal/${slug}.${DEFAULT_LOCALE}.md`]
      return postFrom(slug, raw)
    })
    .sort((a, b) => b.date.localeCompare(a.date))
}

export function journalPost(slug: string, locale: Locale): JournalPost | undefined {
  const raw =
    journalFiles[`./journal/${slug}.${locale}.md`] ??
    journalFiles[`./journal/${slug}.${DEFAULT_LOCALE}.md`]
  return raw === undefined ? undefined : postFrom(slug, raw)
}
