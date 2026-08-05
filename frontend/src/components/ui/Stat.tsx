import { cx } from '../../lib/cx'

/**
 * "1,400 m / Meadow altitude" — the hero's credibility strip, reused by the
 * Hive club panel on the sign-in screen.
 *
 * <dl> rather than two <div>s: these are term/definition pairs, and the
 * markup saying so is free. Each Stat renders its own <dl> so callers can
 * lay them out in any container (flex row, grid) without the list structure
 * dictating the layout.
 */
export function Stat({
  value,
  label,
  tone = 'light',
  className,
}: {
  value: string
  label: string
  tone?: 'light' | 'dark'
  className?: string
}) {
  return (
    <dl className={cx('flex flex-col gap-0.5', className)}>
      <dd
        className={cx(
          'font-display text-display-xs font-extrabold',
          tone === 'dark' ? 'text-honey' : 'text-ink',
        )}
      >
        {value}
      </dd>
      <dt
        className={cx(
          'text-xs',
          tone === 'dark' ? 'text-ink-on-dark-soft' : 'text-ink-muted',
        )}
      >
        {label}
      </dt>
    </dl>
  )
}
