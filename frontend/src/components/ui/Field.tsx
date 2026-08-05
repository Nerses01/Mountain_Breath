import type { ReactNode } from 'react'
import { describedById } from './fieldStyles'

/**
 * The label / control / message shell shared by every form control.
 *
 * The accessibility wiring lives here so no individual field can forget it:
 * `htmlFor` ties the label to the control (clicking the label focuses it,
 * and a screen reader announces the two together), and the message gets a
 * predictable id that the control points at with `aria-describedby`. The
 * design's forms show a bare label above a box and nothing else, so all of
 * this is ours to add — E10's audit lists exactly this as the gap.
 */
export function Field({
  id,
  label,
  error,
  hint,
  children,
}: {
  id: string
  label: string
  error?: string
  hint?: string
  children: ReactNode
}) {
  return (
    <div className="flex flex-col gap-2">
      <label htmlFor={id} className="text-xs font-semibold text-ink-soft">
        {label}
      </label>
      {children}
      {error ? (
        // role="alert" makes a screen reader announce the message the moment
        // it appears, rather than only when the user happens to reach it.
        <p id={describedById(id)} role="alert" className="text-xs text-danger">
          {error}
        </p>
      ) : (
        hint && (
          <p id={describedById(id)} className="text-xs text-ink-muted">
            {hint}
          </p>
        )
      )}
    </div>
  )
}
