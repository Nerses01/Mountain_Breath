import { cx } from '../../lib/cx'

export type ButtonVariant = 'primary' | 'dark' | 'honey' | 'outline' | 'ghost'
export type ButtonSize = 'sm' | 'md' | 'lg'

// Every variant the design uses, in one place. `primary` deliberately paints
// itself in --color-brand-ink rather than the mock's brighter --color-brand:
// cream text on the bright orange measures 2.9:1, under the 4.5:1 floor.
// The mock's orange glow under primary buttons (--shadow-cta) was dropped by
// request in Aug 2026 — it read as a smear under the dark checkout card.
const VARIANTS: Record<ButtonVariant, string> = {
  primary: 'bg-brand-ink text-ink-on-dark hover:opacity-90',
  dark: 'bg-bark text-ink-on-dark hover:opacity-90',
  honey: 'bg-honey text-ink hover:opacity-90',
  outline:
    'border-[1.5px] border-bark text-ink hover:bg-bark hover:text-ink-on-dark',
  // Not a pill: the mock's tertiary action is a label underlined in honey.
  ghost: 'text-ink border-b-2 border-honey pb-1 hover:border-brand-ink',
}

const SIZES: Record<ButtonSize, string> = {
  sm: 'px-4 py-2 text-sm',
  md: 'px-5 py-2.5 text-sm',
  lg: 'px-8 py-4 text-base',
}

/**
 * Shared by <Button> and <ButtonLink> so a button and a link that look
 * identical cannot drift apart.
 *
 * Lives in its own module rather than beside the components: a file that
 * exports both components and plain helpers defeats Vite's fast refresh,
 * which can only hot-swap a module when everything it exports is a
 * component.
 */
export function buttonClasses(
  variant: ButtonVariant = 'primary',
  size: ButtonSize = 'md',
  fullWidth = false,
): string {
  return cx(
    'inline-flex items-center justify-center gap-2 font-display font-semibold transition',
    // The ghost variant is underlined text, so the pill radius would draw a
    // rounded box around nothing.
    variant !== 'ghost' && 'rounded-full',
    VARIANTS[variant],
    SIZES[size],
    fullWidth && 'w-full',
    'disabled:pointer-events-none disabled:opacity-60',
  )
}
