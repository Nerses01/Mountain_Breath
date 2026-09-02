import { useEffect, useId, useState } from 'react'
import { formatMoney } from '../../lib/format'
import { useCurrency } from '../../lib/useCurrency'

/**
 * The sidebar's dual price slider.
 *
 * Built from TWO native <input type="range"> elements stacked on one track,
 * not from a div with pointer handlers. A native range comes with keyboard
 * support (arrows, Home/End, Page Up/Down), a screen-reader announcement of
 * its value, and the OS's own touch target sizes — all of which a custom
 * slider has to reimplement and usually gets partly wrong. The cost is that
 * the two thumbs overlap, which is handled below.
 *
 * Three problems a dual slider has that a single one does not:
 *
 *  1. THE THUMBS CAN CROSS. Dragging the low thumb past the high one would
 *     produce min > max. Clamped on change, so the pair can meet but never
 *     swap.
 *  2. THE INPUTS OVERLAP AND SWALLOW CLICKS. Both cover the whole track, so
 *     the one drawn on top would take every press. `pointer-events: none` on
 *     the input and `auto` on its thumb means only the 16px thumbs are
 *     interactive, and the rest of the track passes clicks through.
 *  3. THE FILTER MUST NOT FIRE PER PIXEL. Dragging a range emits a change
 *     for every step; each one would be a new query. The value is held in
 *     local state while dragging and committed on release (onChange fires
 *     continuously, but pointer/key release is what calls onCommit), so a
 *     drag from $9 to $32 is one request rather than twenty-three.
 */
export function PriceRange({
  min,
  max,
  value,
  onCommit,
  label,
}: {
  /** Bounds in minor units, from the facets endpoint. */
  min: number
  max: number
  /** Current selection, clamped into [min, max] on arrival. */
  value: { min: number; max: number }
  onCommit: (next: { min: number; max: number }) => void
  label: string
}) {
  const id = useId()
  const [draft, setDraft] = useState(value)
  // The bounds arrive from the facets endpoint already denominated in the
  // shopper's market — the slider never converts, it just labels.
  const { currency } = useCurrency()

  // The URL is the source of truth: back/forward and a shared link change
  // `value` from the outside, and the thumbs have to follow. Without this
  // the slider would keep showing whatever it was last dragged to.
  useEffect(() => setDraft(value), [value.min, value.max]) // eslint-disable-line react-hooks/exhaustive-deps

  // A one-product catalog has min === max, which would make the range's step
  // arithmetic divide by zero below. Guard rather than hide the control:
  // a disabled slider still says "this is the price, and there is only one".
  const span = Math.max(max - min, 1)
  const lowPct = ((draft.min - min) / span) * 100
  const highPct = ((draft.max - min) / span) * 100

  // Step in whole currency units (100 minor) — nobody filters by the cent,
  // and a 1-minor-unit step would make the keyboard arrows useless.
  const step = 100

  const commit = () => onCommit(draft)

  return (
    <div className="flex flex-col gap-3">
      <div className="relative h-4">
        {/* The track and the selected span. aria-hidden: the two inputs
            below already announce the values, and a screen reader reading
            the decoration too would just add noise. */}
        <span
          aria-hidden
          className="absolute inset-x-0 top-1.5 h-1 rounded-full bg-line"
        />
        <span
          aria-hidden
          className="absolute top-1.5 h-1 rounded-full bg-brand"
          style={{ left: `${lowPct}%`, right: `${100 - highPct}%` }}
        />

        <input
          type="range"
          id={`${id}-min`}
          aria-label={`${label} — minimum`}
          min={min}
          max={max}
          step={step}
          value={draft.min}
          disabled={max <= min}
          onChange={(e) =>
            setDraft((d) => ({ ...d, min: Math.min(Number(e.target.value), d.max) }))
          }
          onPointerUp={commit}
          onKeyUp={commit}
          onBlur={commit}
          className="range-thumb absolute inset-x-0 top-0 h-4 w-full appearance-none bg-transparent"
        />
        <input
          type="range"
          id={`${id}-max`}
          aria-label={`${label} — maximum`}
          min={min}
          max={max}
          step={step}
          value={draft.max}
          disabled={max <= min}
          onChange={(e) =>
            setDraft((d) => ({ ...d, max: Math.max(Number(e.target.value), d.min) }))
          }
          onPointerUp={commit}
          onKeyUp={commit}
          onBlur={commit}
          className="range-thumb absolute inset-x-0 top-0 h-4 w-full appearance-none bg-transparent"
        />
      </div>

      <p className="flex justify-between text-sm text-ink-body">
        <span>{formatMoney(draft.min, currency)}</span>
        <span>{formatMoney(draft.max, currency)}</span>
      </p>
    </div>
  )
}
