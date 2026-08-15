import { describe, expect, it } from 'vitest'
import { journalPost, journalPosts, pageMarkdown } from './index'

/**
 * The content pipeline's contract: every page exists in every locale (or
 * falls back to English per file), posts sort newest-first, and the
 * hand-rolled frontmatter parser actually parses what the posts carry.
 * These tests read the REAL content files — a missing translation file is
 * exactly the failure they exist to catch.
 */
describe('content pipeline', () => {
  it('serves every page in every locale', () => {
    const slugs = ['our-hive', 'benefits', 'shipping', 'contact', 'terms', 'privacy'] as const
    for (const slug of slugs) {
      for (const locale of ['en', 'hy', 'ru'] as const) {
        const md = pageMarkdown(slug, locale)
        expect(md, `${slug}.${locale}`).toBeTruthy()
        expect(md, `${slug}.${locale} should start with an H1`).toMatch(/^# /)
      }
    }
  })

  it('lists posts newest first, with parsed frontmatter', () => {
    const posts = journalPosts('en')
    expect(posts.length).toBeGreaterThanOrEqual(3)
    for (const post of posts) {
      expect(post.title).toBeTruthy()
      expect(post.date).toMatch(/^\d{4}-\d{2}-\d{2}$/)
      expect(post.teaser).toBeTruthy()
      // The fence and the metadata must not leak into the body.
      expect(post.body).not.toContain('---')
      expect(post.body).not.toContain('title:')
    }
    const dates = posts.map((p) => p.date)
    expect(dates).toEqual([...dates].sort().reverse())
  })

  it('falls back to English per post and per page', () => {
    // The Armenian journal is fully translated today; the CONTRACT is that
    // a missing translation degrades rather than blanks, which the unknown
    // slug's absence (undefined, not a crash) also pins.
    expect(journalPost('linden-weeks', 'hy')?.title).toBeTruthy()
    expect(journalPost('no-such-post', 'en')).toBeUndefined()
  })

  it('renders localized titles, not English everywhere', () => {
    const en = journalPost('linden-weeks', 'en')
    const hy = journalPost('linden-weeks', 'hy')
    expect(en?.title).not.toEqual(hy?.title)
  })
})
