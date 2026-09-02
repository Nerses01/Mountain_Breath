import { useTranslation } from 'react-i18next'
import { cx } from '../../lib/cx'

/**
 * Numbered pages plus a next arrow — the design's pagination, shared by the
 * shop grid and the order history. A `nav` landmark so assistive tech can
 * jump straight to it, aria-current on the page you are on.
 *
 * Fine for the page counts this shop sees (a catalog of six, a history of
 * a dozen); a hundred pages would need windowing (1 … 4 5 6 … 100) before
 * this row becomes a wall of buttons.
 */
export function Pagination({
  page,
  pageCount,
  onSelect,
}: {
  page: number
  pageCount: number
  onSelect: (page: number) => void
}) {
  const { t } = useTranslation()
  const pages = Array.from({ length: pageCount }, (_, i) => i + 1)

  return (
    <nav aria-label={t('catalog:pagination')} className="flex justify-center gap-2.5 pt-2">
      {pages.map((n) => (
        <button
          key={n}
          type="button"
          aria-current={n === page ? 'page' : undefined}
          onClick={() => onSelect(n)}
          className={cx(
            'inline-flex size-9.5 items-center justify-center rounded-full font-display text-sm transition',
            n === page
              ? 'bg-brand-ink font-bold text-ink-on-dark'
              : 'border-[1.5px] border-line font-semibold text-ink-body hover:border-line-strong',
          )}
        >
          {n}
        </button>
      ))}
      <button
        type="button"
        aria-label={t('catalog:nextPage')}
        disabled={page >= pageCount}
        onClick={() => onSelect(page + 1)}
        className="inline-flex size-9.5 items-center justify-center rounded-full border-[1.5px] border-line text-sm text-ink-body transition hover:border-line-strong disabled:opacity-40"
      >
        →
      </button>
    </nav>
  )
}
