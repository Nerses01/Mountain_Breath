import { cx } from '../../lib/cx'

/**
 * The canvas's toggle (A5) as a real switch: a BUTTON with role="switch"
 * and aria-checked — the ARIA pattern for on/off that takes effect
 * immediately (a checkbox implies a form that submits later). Space and
 * Enter both fire, for free, because it is a button.
 *
 * The label sits in the same component so the control is never rendered
 * nameless; `description` is the canvas's small line underneath, wired via
 * aria-describedby. A disabled switch is the honest-stub rendering — state
 * visible, control inert, the reason stated by the caller next to it.
 */
export function Switch({
  checked,
  onChange,
  label,
  description,
  disabled = false,
}: {
  checked: boolean
  onChange: (next: boolean) => void
  label: string
  description?: string
  disabled?: boolean
}) {
  const descriptionId = description ? `switch-desc-${label.replace(/\s+/g, '-')}` : undefined
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className={cx('text-[0.9375rem] font-bold', disabled ? 'text-ink-muted' : 'text-ink')}>
          {label}
        </span>
        {description && (
          <span id={descriptionId} className="text-[0.8125rem] text-ink-muted">
            {description}
          </span>
        )}
      </span>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        aria-describedby={descriptionId}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={cx(
          'relative h-6 w-10.5 shrink-0 rounded-full transition-colors',
          checked ? 'bg-brand-ink' : 'bg-line',
          disabled && 'opacity-50',
        )}
      >
        <span
          aria-hidden
          className={cx(
            'absolute top-0.75 size-4.5 rounded-full bg-card transition-[left]',
            checked ? 'left-5.25' : 'left-0.75',
          )}
        />
      </button>
    </div>
  )
}
