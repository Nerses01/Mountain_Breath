import { cx } from '../../lib/cx'
import { MinusIcon, PlusIcon } from './icons'

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
 *
 * Those labels are PROPS with English defaults rather than being built from
 * `label` in English ("Decrease " + label), which is what E1 did while this
 * component was still unused. E3 put it on a storefront page, where an
 * Armenian reader would have met two English button names — a string is no
 * less hardcoded for being assembled at runtime.
 */
export function QtyStepper({
  value,
  onChange,
  min = 1,
  max,
  label = 'Quantity',
  decreaseLabel,
  increaseLabel,
  className,
}: {
  value: number
  onChange: (next: number) => void
  min?: number
  max?: number
  label?: string
  decreaseLabel?: string
  increaseLabel?: string
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
        aria-label={decreaseLabel ?? `Decrease ${label.toLowerCase()}`}
        disabled={atMin}
        onClick={() => onChange(clamp(value - 1))}
        className="text-ink-muted disabled:opacity-40"
      >
        <MinusIcon size={16} />
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
        aria-label={increaseLabel ?? `Increase ${label.toLowerCase()}`}
        disabled={atMax}
        onClick={() => onChange(clamp(value + 1))}
        className="text-ink disabled:opacity-40"
      >
        <PlusIcon size={16} />
      </button>
    </div>
  )
}
