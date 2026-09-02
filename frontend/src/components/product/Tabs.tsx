import { useEffect, useRef, useState, type ReactNode } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { cx } from '../../lib/cx'

export type Tab = {
  /** Goes in the URL hash, so keep it short and stable. */
  id: string
  label: string
  panel: ReactNode
}

/**
 * The product page's "How to take it · Storage" tabs, following the ARIA
 * tabs pattern — and putting the active tab in the URL HASH so a tab is a
 * linkable thing.
 *
 * Why the hash and not a query param: a tab is a position WITHIN a document,
 * which is exactly what a fragment identifier means. It also costs no server
 * round trip and does not collide with the shop's filter params.
 *
 * The keyboard contract, same as the gallery's and for the same reason —
 * one tab stop for the whole strip:
 *
 *   Tab        moves INTO the strip, then straight on to the panel
 *   ← →        move between tabs and select as they go
 *   Home/End   first / last tab
 *
 * The panel itself is `tabIndex={0}`: its content may be plain text with
 * nothing focusable in it, and without this a keyboard user could select a
 * tab but never scroll its contents.
 */
export function Tabs({ tabs, label }: { tabs: Tab[]; label: string }) {
  const { hash } = useLocation()
  const navigate = useNavigate()
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([])
  const shouldFocus = useRef(false)

  // The URL is the source of truth, with local state only as the fallback
  // for "no hash yet" — the same one-direction rule the shop's filters
  // follow, so back/forward move between tabs.
  const fromHash = tabs.findIndex((t) => `#${t.id}` === hash)
  const [fallback, setFallback] = useState(0)
  const active = fromHash >= 0 ? fromHash : fallback

  useEffect(() => {
    if (shouldFocus.current) {
      tabRefs.current[active]?.focus()
      shouldFocus.current = false
    }
  }, [active])

  const select = (index: number, viaKeyboard = false) => {
    const next = (index + tabs.length) % tabs.length
    shouldFocus.current = viaKeyboard
    setFallback(next)
    // replace: true — walking three tabs should not put three entries in
    // the history and make the back button feel broken.
    navigate(`#${tabs[next].id}`, { replace: true })
  }

  const onKeyDown = (e: React.KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowRight':
        e.preventDefault()
        select(active + 1, true)
        break
      case 'ArrowLeft':
        e.preventDefault()
        select(active - 1, true)
        break
      case 'Home':
        e.preventDefault()
        select(0, true)
        break
      case 'End':
        e.preventDefault()
        select(tabs.length - 1, true)
        break
    }
  }

  return (
    <div className="flex flex-col gap-5.5">
      <div
        role="tablist"
        aria-label={label}
        onKeyDown={onKeyDown}
        className="flex gap-7.5 border-b-[1.5px] border-line"
      >
        {tabs.map((tab, i) => (
          <button
            key={tab.id}
            ref={(el) => {
              tabRefs.current[i] = el
            }}
            id={`tab-${tab.id}`}
            type="button"
            role="tab"
            aria-selected={i === active}
            aria-controls={`panel-${tab.id}`}
            tabIndex={i === active ? 0 : -1}
            onClick={() => select(i)}
            className={cx(
              '-mb-px border-b-[3px] pb-3.5 font-display text-base transition',
              i === active
                ? 'border-brand font-bold text-ink'
                : 'border-transparent font-medium text-ink-muted hover:text-ink',
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {tabs.map((tab, i) => (
        <div
          key={tab.id}
          id={`panel-${tab.id}`}
          role="tabpanel"
          aria-labelledby={`tab-${tab.id}`}
          hidden={i !== active}
          tabIndex={0}
        >
          {tab.panel}
        </div>
      ))}
    </div>
  )
}
