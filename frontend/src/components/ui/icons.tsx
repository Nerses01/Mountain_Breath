import type { ReactNode } from 'react'

/**
 * The design's icon set as inline components.
 *
 * Chosen over an SVG sprite (`<use href="/icons.svg#search">`) for three
 * reasons: `currentColor` below means an icon simply inherits whatever text
 * colour the token system already put on its parent, so `text-ink-muted` on
 * an IconButton colours the glyph with no extra wiring; unused icons drop out
 * of the bundle; and there is no second HTTP request, nor the cross-document
 * styling quirks that external sprites still carry in some browsers.
 *
 * All icons are decorative — `aria-hidden` here, with the accessible name
 * coming from the control that wraps them (see IconButton's required `label`
 * prop). `focusable="false"` keeps them out of the tab order.
 */
type IconProps = {
  /** Rendered size in px, square. */
  size?: number
  className?: string
}

function IconBase({
  size = 20,
  className,
  filled = false,
  children,
}: IconProps & { filled?: boolean; children: ReactNode }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill={filled ? 'currentColor' : 'none'}
      stroke="currentColor"
      strokeWidth={1.75}
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden
      focusable="false"
    >
      {children}
    </svg>
  )
}

export function SearchIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.8-3.8" />
    </IconBase>
  )
}

export function HeartIcon({ filled = false, ...props }: IconProps & { filled?: boolean }) {
  // Filled is the "saved" state on wishlist toggles (E8); the same path
  // serves both so the shape does not shift when it fills in.
  return (
    <IconBase {...props} filled={filled}>
      <path d="M20.8 4.6a5.5 5.5 0 0 0-7.8 0L12 5.7l-1-1.1a5.5 5.5 0 1 0-7.8 7.8l1 1.1L12 21l7.8-7.5 1-1.1a5.5 5.5 0 0 0 0-7.8z" />
    </IconBase>
  )
}

export function ArrowRightIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M5 12h14M13 6l6 6-6 6" />
    </IconBase>
  )
}

export function ChevronDownIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="m6 9 6 6 6-6" />
    </IconBase>
  )
}

export function MinusIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M5 12h14" />
    </IconBase>
  )
}

export function PlusIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M12 5v14M5 12h14" />
    </IconBase>
  )
}

export function CheckIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="m4.5 12.5 5 5L19.5 7" />
    </IconBase>
  )
}

export function StarIcon({ filled = false, ...props }: IconProps & { filled?: boolean }) {
  return (
    <IconBase {...props} filled={filled}>
      <path d="m12 3.2 2.85 5.78 6.38.93-4.62 4.5 1.09 6.35L12 17.76l-5.7 3-1.09-6.35-4.62-4.5 6.38-.93z" />
    </IconBase>
  )
}
