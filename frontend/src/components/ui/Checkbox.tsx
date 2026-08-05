import { type InputHTMLAttributes } from 'react'
import { cx } from '../../lib/cx'
import { CheckIcon } from './icons'

/**
 * The design draws a filled orange square with a check glyph. Rather than
 * hiding the real control behind a <div> and rebuilding its behaviour, the
 * native <input type="checkbox"> stays in the DOM but visually hidden
 * (`sr-only`), and a sibling <span> is painted from its state via Tailwind's
 * `peer-*` variants.
 *
 * That keeps the whole native contract for free — space toggles it, it takes
 * part in form submission, screen readers announce "checkbox, checked", and
 * the global :focus-visible ring from index.css still applies through
 * peer-focus-visible.
 *
 * Note the check itself is toggled with text COLOUR, not a nested element:
 * `peer-*` compiles to a sibling combinator (`.peer:checked ~ .target`), so
 * it only ever reaches the span next to the input — never anything nested
 * inside it.
 */
export function Checkbox({
  label,
  className,
  ...rest
}: InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  return (
    <label className={cx('inline-flex cursor-pointer items-center gap-3', className)}>
      <input type="checkbox" className="peer sr-only" {...rest} />
      <span
        // aria-hidden: this is decoration. The real checkbox above is what
        // assistive tech reads, so announcing the box twice would be noise.
        aria-hidden
        className={cx(
          'flex size-5 shrink-0 items-center justify-center rounded-sm',
          'border-[1.5px] border-line-strong bg-card',
          // The tick is drawn with currentColor, so making the text
          // transparent hides it without a second toggle to keep in sync.
          'text-transparent',
          'peer-checked:border-brand-ink peer-checked:bg-brand-ink',
          'peer-checked:text-ink-on-dark',
          'peer-focus-visible:outline-2 peer-focus-visible:outline-offset-2',
          'peer-focus-visible:outline-brand-ink',
        )}
      >
        <CheckIcon size={13} />
      </span>
      <span className="text-sm text-ink-body">{label}</span>
    </label>
  )
}
