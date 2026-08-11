import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useProducts } from '../../api/hooks'
import { useLocale } from '../../i18n/useLocale'
import { useDebouncedValue } from '../../lib/useDebounce'
import { Price } from '../ui/Price'
import { SearchIcon } from '../ui'

/**
 * Search, moved out of the catalog body and into a header overlay (E2).
 *
 * It keeps Era I's behaviour exactly: the input updates on every keystroke
 * so typing stays responsive, but the QUERY is debounced by 300 ms, so a
 * six-letter word is one request instead of six. The backend's three search
 * doors — full text, substring, trigram similarity — are unchanged, which is
 * why "hony" still finds honey.
 *
 * Accessibility, none of which the mock draws (§6 exception 2):
 *  - role="dialog" + aria-modal so a screen reader treats the page behind it
 *    as inert rather than reading straight through it.
 *  - Escape closes, and focus RETURNS to the button that opened it. Losing
 *    focus to <body> on close is the classic modal bug: the next Tab starts
 *    from the top of the page.
 *  - The results are a <ul> with aria-live, so their arrival is announced.
 */
export function SearchOverlay({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const navigate = useNavigate()

  const [term, setTerm] = useState('')
  const debounced = useDebouncedValue(term, 300)
  const inputRef = useRef<HTMLInputElement>(null)

  // A one-character query matches almost everything, so it is not worth a
  // round trip. `enabled` is how TanStack Query expresses "not yet" — the
  // hook still runs (rules of hooks), the request does not.
  const active = debounced.trim().length >= 2
  const results = useProducts({ q: debounced, perPage: 5 }, active)

  useEffect(() => {
    inputRef.current?.focus()
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [onClose])

  const submit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!term.trim()) return
    // Enter hands the query to the Shop page, where it becomes part of the
    // URL like every other filter — so a search is shareable and survives a
    // reload, instead of living in this component's state.
    navigate(`${localePath('/shop')}?q=${encodeURIComponent(term.trim())}`)
    onClose()
  }

  return (
    <div
      className="fixed inset-0 z-50 flex justify-center bg-bark/40 px-6 pt-24"
      // A click on the backdrop closes; a click inside must not. Comparing
      // target to currentTarget is what distinguishes the two without
      // stopPropagation, which would break unrelated listeners.
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={t('common:actions.search')}
        className="h-fit w-full max-w-160 rounded-2xl bg-card p-6 shadow-screen"
      >
        <form onSubmit={submit} className="flex items-center gap-3">
          <SearchIcon className="size-5 shrink-0 text-ink-muted" />
          <input
            ref={inputRef}
            type="search"
            value={term}
            onChange={(e) => setTerm(e.target.value)}
            placeholder={t('catalog:searchPlaceholder')}
            aria-label={t('common:actions.search')}
            className="min-w-0 flex-1 bg-transparent py-2 text-base text-ink outline-none placeholder:text-ink-muted"
          />
          <button
            type="button"
            onClick={onClose}
            className="shrink-0 font-display text-sm font-semibold text-ink-muted hover:text-ink"
          >
            {t('common:actions.close')}
          </button>
        </form>

        {active && (
          <div className="mt-4 border-t border-line-soft pt-4">
            {results.isPending && (
              <p className="text-sm text-ink-muted">{t('common:state.loading')}</p>
            )}

            {results.data && results.data.total === 0 && (
              <p className="text-sm text-ink-soft">
                {t('catalog:noResultsFor', { query: debounced })}
              </p>
            )}

            <ul aria-live="polite" className="flex flex-col">
              {results.data?.items.map((p) => (
                <li key={p.id}>
                  <Link
                    to={localePath(`/products/${p.slug}`)}
                    onClick={onClose}
                    className="flex items-center justify-between gap-4 rounded-md px-2 py-2.5 hover:bg-panel"
                  >
                    <span className="font-display text-sm font-semibold text-ink">
                      {p.name}
                    </span>
                    <span className="shrink-0 text-sm text-ink-muted">
                      {p.variants[0] && (
                        <Price
                          prices={p.variants[0].prices}
                          primaryMinor={p.variants[0].price_minor}
                          size="sm"
                        />
                      )}
                    </span>
                  </Link>
                </li>
              ))}
            </ul>

            {(results.data?.total ?? 0) > 0 && (
              <Link
                to={`${localePath('/shop')}?q=${encodeURIComponent(debounced.trim())}`}
                onClick={onClose}
                className="mt-2 inline-block px-2 font-display text-sm font-semibold text-brand-ink hover:underline"
              >
                {t('catalog:seeAllResults', { count: results.data?.total ?? 0 })} →
              </Link>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
