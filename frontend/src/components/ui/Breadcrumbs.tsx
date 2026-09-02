import { Link } from 'react-router'

export type Crumb = { label: string; to?: string }

/**
 * "Home / Shop / Royal jelly / Fresh Royal Jelly".
 *
 * Structure matters more than it looks: a <nav> with an accessible name lets
 * a screen reader jump straight here, the <ol> says these steps are ordered
 * rather than an arbitrary pile of links, and aria-current="page" marks which
 * one you are actually on. The separators are aria-hidden — a slash read
 * aloud between every item is noise.
 *
 * Note the muted text uses --color-ink-muted, not the mock's #a9714b: at
 * 13px that measures 3.6:1 and fails AA (see the token block in index.css).
 */
export function Breadcrumbs({ items }: { items: Crumb[] }) {
  return (
    <nav aria-label="Breadcrumb">
      <ol className="flex flex-wrap items-center gap-2 text-xs text-ink-muted">
        {items.map((item, i) => {
          const isLast = i === items.length - 1
          return (
            <li key={`${item.label}-${i}`} className="flex items-center gap-2">
              {item.to && !isLast ? (
                <Link to={item.to} className="hover:text-brand-ink">
                  {item.label}
                </Link>
              ) : (
                <span
                  className={isLast ? 'text-ink' : undefined}
                  aria-current={isLast ? 'page' : undefined}
                >
                  {item.label}
                </span>
              )}
              {!isLast && <span aria-hidden>/</span>}
            </li>
          )
        })}
      </ol>
    </nav>
  )
}
