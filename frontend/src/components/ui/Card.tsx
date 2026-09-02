import type { HTMLAttributes } from 'react'
import { cx } from '../../lib/cx'

export type CardTone = 'card' | 'panel' | 'bark' | 'honey'

const TONES: Record<CardTone, string> = {
  card: 'bg-card text-ink-body',
  panel: 'bg-panel text-ink-body',
  // On bark, the default body ink would be unreadable — each dark surface
  // brings its own text colour rather than leaving callers to remember.
  bark: 'bg-bark text-ink-on-dark-body',
  honey: 'bg-honey text-ink-strong',
}

/**
 * The design's recurring rounded block. Padding is a prop rather than baked
 * in because the mock uses the same surface at 18px (product cards), 24px
 * (sidebar filters) and 34px (the "How we harvest" panel).
 */
export function Card({
  tone = 'card',
  padded = true,
  className,
  ...rest
}: HTMLAttributes<HTMLDivElement> & {
  tone?: CardTone
  padded?: boolean
}) {
  return (
    <div
      className={cx('rounded-xl', TONES[tone], padded && 'p-6', className)}
      {...rest}
    />
  )
}
