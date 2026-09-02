import { useId, type InputHTMLAttributes } from 'react'
import { cx } from '../../lib/cx'
import { Field } from './Field'
import { controlClasses, describedById } from './fieldStyles'

export function Input({
  label,
  error,
  hint,
  id,
  className,
  ...rest
}: InputHTMLAttributes<HTMLInputElement> & {
  label: string
  error?: string
  hint?: string
}) {
  // useId generates a collision-free id per instance, so a page can render
  // three <Input label="Email"> without their labels pointing at the same
  // control. Callers may still pass an explicit id when they need to target
  // it (tests, a "skip to field" link).
  const generatedId = useId()
  const fieldId = id ?? generatedId
  const hasMessage = Boolean(error || hint)

  return (
    <Field id={fieldId} label={label} error={error} hint={hint}>
      <input
        id={fieldId}
        // aria-invalid tells assistive tech the value was rejected; the
        // red border alone conveys that to sighted users only.
        aria-invalid={error ? true : undefined}
        aria-describedby={hasMessage ? describedById(fieldId) : undefined}
        className={cx(controlClasses(Boolean(error)), className)}
        {...rest}
      />
    </Field>
  )
}
