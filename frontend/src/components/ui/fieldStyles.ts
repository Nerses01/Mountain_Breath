import { cx } from '../../lib/cx'

/**
 * The id a control points at with `aria-describedby`, derived from the
 * field's own id so the control and its message can never disagree about
 * what to call it.
 */
export function describedById(id: string): string {
  return `${id}-message`
}

/** Shared look for the box itself — <input>, <select> and friends. */
export function controlClasses(hasError = false): string {
  return cx(
    'w-full rounded-md border-[1.5px] bg-card px-4 py-3',
    'text-base text-ink placeholder:text-ink-muted',
    hasError ? 'border-danger' : 'border-line',
  )
}
