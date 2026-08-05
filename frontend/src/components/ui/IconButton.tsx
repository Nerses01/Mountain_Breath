import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { cx } from '../../lib/cx'

/**
 * The 38px circular control from the header — search, wishlist, and the
 * gallery arrows later on.
 *
 * `label` is REQUIRED, not optional. An icon-only button renders no text, so
 * without an accessible name a screen reader announces just "button" and the
 * user has no idea what it does. Making the prop mandatory means the type
 * checker refuses to let us ship that — the compiler enforcing an
 * accessibility rule rather than a code review having to catch it.
 */
export function IconButton({
  label,
  tone = 'outline',
  className,
  children,
  type = 'button',
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  label: string
  tone?: 'outline' | 'bare'
  children: ReactNode
}) {
  return (
    <button
      type={type}
      aria-label={label}
      title={label}
      className={cx(
        'inline-flex size-9.5 items-center justify-center rounded-full',
        'text-ink-muted transition hover:text-ink',
        tone === 'outline' && 'border-[1.5px] border-line-strong',
        'disabled:pointer-events-none disabled:opacity-60',
        className,
      )}
      {...rest}
    >
      {/* The glyph is decoration — `label` above is the accessible name, so
          announcing the icon too would just repeat it. */}
      <span aria-hidden>{children}</span>
    </button>
  )
}
