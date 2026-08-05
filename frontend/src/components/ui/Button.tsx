import type { ButtonHTMLAttributes, ReactNode } from 'react'
import { Link } from 'react-router'
import { cx } from '../../lib/cx'
import { buttonClasses, type ButtonSize, type ButtonVariant } from './buttonStyles'

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant
  size?: ButtonSize
  fullWidth?: boolean
}

export function Button({
  variant = 'primary',
  size = 'md',
  fullWidth = false,
  className,
  type = 'button',
  ...rest
}: ButtonProps) {
  // Defaulting to type="button" matters: an unqualified <button> inside a
  // <form> defaults to type="submit" and will submit it on click. That is a
  // classic accidental-submit bug, so opting in is the safer default.
  return (
    <button
      type={type}
      className={cx(buttonClasses(variant, size, fullWidth), className)}
      {...rest}
    />
  )
}

/**
 * A router link wearing the same clothes. Kept as a separate component
 * rather than a `to?: string` prop on <Button> because the two render
 * genuinely different elements — a <button> is for actions, an <a> is for
 * navigation, and screen readers and middle-click behave differently. A
 * union prop would have blurred that at the call site.
 */
export function ButtonLink({
  to,
  variant = 'primary',
  size = 'md',
  fullWidth = false,
  className,
  children,
}: {
  to: string
  variant?: ButtonVariant
  size?: ButtonSize
  fullWidth?: boolean
  className?: string
  children: ReactNode
}) {
  return (
    <Link
      to={to}
      className={cx(buttonClasses(variant, size, fullWidth), className)}
    >
      {children}
    </Link>
  )
}
