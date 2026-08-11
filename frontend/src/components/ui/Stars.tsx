import { useTranslation } from 'react-i18next'
import { cx } from '../../lib/cx'
import { StarIcon } from './icons'

/**
 * ★★★★☆ — one implementation, used by the card, the product detail and every
 * review row.
 *
 * HOW THE PARTIAL STAR WORKS. Rendering "4.67 out of 5" as whole and half
 * stars means rounding, and rounding 4.67 to 4.5 puts a visible lie on the
 * page. Instead this draws the five outlines once, then draws five FILLED
 * stars on top inside a box clipped to 93.4% of the width — so the fill lands
 * wherever the average actually is. No per-star branching, no rounding, and
 * any precision the backend chooses renders correctly.
 *
 * ACCESSIBILITY. Ten star glyphs announced one by one are noise, so the whole
 * thing is a single `role="img"` with an aria-label that reads the number —
 * "4.7 out of 5, 12 reviews" — and the glyphs themselves are hidden. That is
 * the pattern for "a picture made of text": name the picture, hide the parts.
 *
 * The design draws ★★★★★ with no partial state and no empty state (§6
 * exception 2), so both are ours.
 */
export function Stars({
  rating,
  count,
  size = 'sm',
  showCount = true,
  className,
}: {
  /** 0–5. Zero means "not rated yet", which renders as empty outlines. */
  rating: number
  /** How many reviews the average is made of. */
  count?: number
  size?: 'sm' | 'md'
  showCount?: boolean
  className?: string
}) {
  const { t } = useTranslation()

  // Clamp rather than trust: this is display code, and a bad number should
  // produce a wrong-looking star row, not a fill that overflows its box.
  const clamped = Math.min(Math.max(rating, 0), 5)
  // Rounded to two decimals because (4.67 / 5) * 100 is 93.39999999999999 in
  // binary floating point, and that whole string would go into the style
  // attribute. The same reason money is stored in minor units, one step down
  // in severity: a sub-pixel difference nobody can see, spelled out in
  // sixteen digits on every card.
  const pct = Math.round((clamped / 5) * 10000) / 100

  const starSize = size === 'md' ? 18 : 14
  const label =
    count === undefined
      ? t('product:rating.outOf', { rating: clamped.toFixed(1) })
      : t('product:rating.outOfWithCount', { rating: clamped.toFixed(1), count })

  if (count === 0) {
    return (
      <span className={cx('text-xs text-ink-muted', className)}>
        {t('product:rating.none')}
      </span>
    )
  }

  return (
    <span className={cx('inline-flex items-center gap-2', className)}>
      <span
        role="img"
        aria-label={label}
        className="relative inline-flex shrink-0"
      >
        {/* The empty row, always five. aria-hidden on both layers: the
            wrapper above is the accessible name. */}
        <span aria-hidden className="flex text-line-strong">
          {[0, 1, 2, 3, 4].map((i) => (
            <StarIcon key={i} size={starSize} />
          ))}
        </span>

        {/* The filled row, clipped to the average. `overflow-hidden` on a
            box of width `pct` is what produces a partial star — the glyph
            underneath is simply cut off mid-shape. */}
        <span
          aria-hidden
          className="absolute inset-y-0 left-0 flex overflow-hidden text-honey"
          style={{ width: `${pct}%` }}
        >
          {[0, 1, 2, 3, 4].map((i) => (
            <StarIcon key={i} size={starSize} filled />
          ))}
        </span>
      </span>

      {showCount && count !== undefined && (
        <span className="text-xs text-ink-muted">
          {t('product:rating.count', { count })}
        </span>
      )}
    </span>
  )
}
