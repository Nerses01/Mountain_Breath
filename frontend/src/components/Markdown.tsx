import { useMemo } from 'react'
import { marked } from 'marked'

/**
 * Renders repo-authored markdown into the design's prose styles.
 *
 * dangerouslySetInnerHTML without a sanitizer is a decision, not an
 * oversight, and this comment is its record: every string that reaches this
 * component comes from `src/content/*.md` — files in THIS repository,
 * written by whoever can already commit code. Sanitizing them would defend
 * the site against its own developers, who could more easily edit the
 * component itself. The moment content comes from a DATABASE or a form,
 * this reasoning dies and DOMPurify walks in — that is the tripwire.
 */
export function Markdown({ markdown }: { markdown: string }) {
  // marked.parse is fast, but re-parsing on every render of a page that
  // also holds a newsletter form's keystrokes would be wasteful.
  const html = useMemo(() => marked.parse(markdown, { async: false }), [markdown])
  return <div className="prose" dangerouslySetInnerHTML={{ __html: html }} />
}
