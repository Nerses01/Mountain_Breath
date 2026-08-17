import { useTranslation } from 'react-i18next'
import { cx } from '../../lib/cx'
import { StarIcon } from './icons'

/**
 * ★★★★☆ — one implementation, used by the card, the product detail and every
 * review row.
 *
 * HOW THE PARTIAL STAR WORKS. Rendering "4.67 out of 5" as whole and half
 * stars means rounding, and rounding 4.67 to 4.5 puts a visible lie on the
 * page. Instead this draws five pale filled stars once, then five dark
 * filled stars on top inside a box clipped to the average — so the fill
 * lands wherever the average actually is. No per-star branching, no
 * rounding, and any precision the backend chooses renders correctly.
 *
 * THE TRAP THIS ONCE FELL INTO: the clipped box is narrower than the five
 * SVGs it contains, and flex items may SHRINK. Instead of the last star
 * being cut off by `overflow-hidden`, all five compressed to fit the box —
 * five squeezed stars sliding out of phase with the row underneath, which
 * read as "sometimes the stars render broken". `shrink-0` on every glyph is
 * what makes the overflow real so the clip has something to cut. The same
 * rule as a fixed-size buffer: the content must keep its size for a
 * truncation to mean anything. Both layers also use the SAME filled glyph,
 * so sub-pixel snapping cannot make them visibly disagree.
 *
 * ACCESSIBILITY. Ten star glyphs announced one by one are noise, so the whole
 * thing is a single `role="img"` with an aria-label that reads the number —
 * "4.7 out of 5, 12 reviews" — and the glyphs themselves are hidden. That is
 * the pattern for "a picture made of text": name the picture, hide the parts.
 *
 * The canvas draws ★★★★★ solid in #46281C — the bark token — which is what
 * the filled layer uses. It has no partial and no empty state (§6 exception
 * 2), so both are ours: the pale solid star underneath is "empty".
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

  const starSize = size === 'md' ? 18 : 14
  // One star's width per rating point, rounded to two decimals because
  // 4.67 × 14 is 65.38000000000001 in binary floating point, and that whole
  // string would go into the style attribute. The same reason money is
  // stored in minor units, one step down in severity.
  const fillPx = Math.round(clamped * starSize * 100) / 100
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
        {/* The "empty" row, always five, solid but pale. aria-hidden on
            both layers: the wrapper above is the accessible name. */}
        <span aria-hidden className="flex text-line-strong">
          {[0, 1, 2, 3, 4].map((i) => (
            <StarIcon key={i} size={starSize} filled className="shrink-0" />
          ))}
        </span>

        {/* The dark row, clipped to the average. `overflow-hidden` on a
            box `fillPx` wide is what produces a partial star — the glyph
            underneath is simply cut off mid-shape. */}
        <span
          aria-hidden
          className="absolute inset-y-0 left-0 flex overflow-hidden text-bark"
          style={{ width: `${fillPx}px` }}
        >
          {[0, 1, 2, 3, 4].map((i) => (
            <StarIcon key={i} size={starSize} filled className="shrink-0" />
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
