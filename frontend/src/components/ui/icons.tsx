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

/**
 * Not in the design — the mock's header carries search, wishlist and cart
 * but no account control, because it never draws a signed-in state. The app
 * has auth, so users need a way to reach it (§6 exception 2: states the mock
 * does not draw are ours to design).
 */
export function UserIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <circle cx="12" cy="8" r="4" />
      <path d="M4 20a8 8 0 0 1 16 0" />
    </IconBase>
  )
}

// E10: the mobile nav's disclosure pair. The mock is desktop-only, so a
// collapsed header is a state it never draws — the plainest possible
// hamburger and cross keep it unambiguous.
export function MenuIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M4 6h16M4 12h16M4 18h16" />
    </IconBase>
  )
}

export function XIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M6 6l12 12M18 6L6 18" />
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

// A1 (account canvas): the rail draws its nav with glyph characters
// (▤ ♡ ⌂ ⚙ →). Characters render differently per OS font, so they become
// icons in the same stroke style as the rest of the set. The heart reuses
// HeartIcon above.

/** The rail's "My orders" mark — a receipt-like card with lines. */
export function OrdersIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <rect x="4.5" y="4" width="15" height="16" rx="2" />
      <path d="M8.5 9h7M8.5 13h7M8.5 17h4" />
    </IconBase>
  )
}

/** The rail's "Addresses" mark — the canvas's ⌂. */
export function HomeIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="m3.5 10.5 8.5-7 8.5 7" />
      <path d="M5.5 9.2V20h13V9.2" />
    </IconBase>
  )
}

/** The rail's "Settings" mark — the canvas's ⚙ as hub-and-spokes. */
export function GearIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <circle cx="12" cy="12" r="3.2" />
      <path d="M12 3.8v2.4M12 17.8v2.4M3.8 12h2.4M17.8 12h2.4M6.2 6.2l1.7 1.7M16.1 16.1l1.7 1.7M17.8 6.2l-1.7 1.7M7.9 16.1l-1.7 1.7" />
    </IconBase>
  )
}

/** The rail's "Log out" mark — the door-and-arrow convention, since the
 *  canvas's bare → is ArrowRightIcon and already means "go somewhere". */
export function LogoutIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M9.5 4H6a1.5 1.5 0 0 0-1.5 1.5v13A1.5 1.5 0 0 0 6 20h3.5" />
      <path d="m14.5 8 4 4-4 4M18.5 12h-10" />
    </IconBase>
  )
}

export function ShieldIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M12 3 5 5.6v5.2c0 4.5 3 8.3 7 9.7 4-1.4 7-5.2 7-9.7V5.6z" />
    </IconBase>
  )
}

export function StarIcon({ filled = false, ...props }: IconProps & { filled?: boolean }) {
  // A REGULAR star, not a hand-drawn one: ten vertices alternating between
  // an outer radius of 9.6 and an inner one of 4.56 (the ★ glyph's chunky
  // proportions, ~0.475 — a pentagram's 0.382 reads spiky at 14px), every
  // 36° around (12, 12). Generated, so it is symmetric by construction.
  return (
    <IconBase {...props} filled={filled}>
      <path d="M12 2.4 14.68 8.31 21.13 9.03 16.34 13.41 17.64 19.77 12 16.56 6.36 19.77 7.66 13.41 2.87 9.03 9.32 8.31 Z" />
    </IconBase>
  )
}
