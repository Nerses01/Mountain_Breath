import type { ReactNode } from 'react'
import { cx } from '../../lib/cx'

export type BadgeTone = 'honey' | 'dark' | 'outline'

// The catalog cards carry one badge each — "Best seller", "New", "Cold
// chain". Tone is a presentation choice the catalog data will eventually
// drive (E2 adds products.badge_tone), so the names stay visual, not
// semantic: a "Cold chain" badge is dark because it looks better dark, not
// because coldness means anything to the design system.
const TONES: Record<BadgeTone, string> = {
  honey: 'bg-honey text-ink',
  dark: 'bg-bark text-honey',
  outline: 'border-[1.5px] border-line-strong text-ink-muted',
}

export function Badge({
  tone = 'honey',
  className,
  children,
}: {
  tone?: BadgeTone
  className?: string
  children: ReactNode
}) {
  return (
    <span
      className={cx(
        'inline-flex items-center rounded-full px-2.5 py-1',
        'font-display text-2xs font-bold uppercase tracking-label',
        TONES[tone],
        className,
      )}
    >
      {children}
    </span>
  )
}
