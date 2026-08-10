import type { ComponentPropsWithRef, ReactNode } from 'react'
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
}: ComponentPropsWithRef<'button'> & {
  label: string
  tone?: 'outline' | 'bare'
  children: ReactNode
}) {
  // ComponentPropsWithRef rather than ButtonHTMLAttributes so `ref` is part
  // of the props. In React 19 a function component takes ref as an ordinary
  // prop — forwardRef is no longer needed — but the TYPE still has to admit
  // it, and ButtonHTMLAttributes does not. The header needs one: closing the
  // search overlay puts focus back on the button that opened it.
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
