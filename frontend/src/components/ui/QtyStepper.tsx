import { cx } from '../../lib/cx'

/**
 * The cart / product-page quantity control: − value +.
 *
 * Clamping lives here rather than in each caller so "you cannot order zero"
 * and "you cannot order more than we have" are enforced in exactly one
 * place. The component is CONTROLLED (value in, onChange out) — it holds no
 * state of its own, so the cart page and the product page can each own the
 * number in whatever way suits them.
 *
 * In the mock the − and + are bare text. That leaves a screen-reader user
 * with two unlabelled buttons, so each gets an explicit aria-label here.
 */
export function QtyStepper({
  value,
  onChange,
  min = 1,
  max,
  label = 'Quantity',
  className,
}: {
  value: number
  onChange: (next: number) => void
  min?: number
  max?: number
  label?: string
  className?: string
}) {
  const clamp = (n: number) => {
    const lower = Math.max(n, min)
    return max === undefined ? lower : Math.min(lower, max)
  }

  const atMin = value <= min
  const atMax = max !== undefined && value >= max

  return (
    <div
      className={cx(
        'inline-flex items-center gap-4 rounded-full',
        'border-[1.5px] border-line bg-card px-5 py-3',
        className,
      )}
    >
      <button
        type="button"
        aria-label={`Decrease ${label.toLowerCase()}`}
        disabled={atMin}
        onClick={() => onChange(clamp(value - 1))}
        className="text-lg leading-none text-ink-muted disabled:opacity-40"
      >
        −
      </button>

      {/* aria-live: the number changes without the focus moving, so without
          this a screen reader would never announce the new quantity. */}
      <span
        aria-live="polite"
        className="min-w-4 text-center font-display text-base font-bold text-ink"
      >
        {value}
      </span>

      <button
        type="button"
        aria-label={`Increase ${label.toLowerCase()}`}
        disabled={atMax}
        onClick={() => onChange(clamp(value + 1))}
        className="text-lg leading-none text-ink disabled:opacity-40"
      >
        +
      </button>
    </div>
  )
}
