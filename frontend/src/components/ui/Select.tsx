import { useId, type SelectHTMLAttributes } from 'react'
import { cx } from '../../lib/cx'
import { Field } from './Field'
import { controlClasses, describedById } from './fieldStyles'

export type SelectOption = { value: string; label: string }

/**
 * A native <select> in the design's clothing.
 *
 * Native rather than a custom dropdown on purpose: the browser gives us
 * keyboard support, type-ahead, mobile's native picker and screen-reader
 * semantics for free, all of which a div-based menu has to reimplement and
 * usually gets subtly wrong. The mock's sort control is only a styled box
 * with a chevron, so there is nothing here worth that trade.
 */
export function Select({
  label,
  options,
  error,
  hint,
  id,
  className,
  ...rest
}: SelectHTMLAttributes<HTMLSelectElement> & {
  label: string
  options: SelectOption[]
  error?: string
  hint?: string
}) {
  const generatedId = useId()
  const fieldId = id ?? generatedId
  const hasMessage = Boolean(error || hint)

  return (
    <Field id={fieldId} label={label} error={error} hint={hint}>
      <select
        id={fieldId}
        aria-invalid={error ? true : undefined}
        aria-describedby={hasMessage ? describedById(fieldId) : undefined}
        className={cx(controlClasses(Boolean(error)), 'pr-10', className)}
        {...rest}
      >
        {options.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </Field>
  )
}
