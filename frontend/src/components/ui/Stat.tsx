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
  // E10 axe finds, both in this one component:
  //  - a <dl>'s TERM must precede its definition in the DOM (the axe
  //    definition-list rule); the design draws the value on top, so the DOM
  //    order is dt→dd and flex-col-reverse restores the visual.
  //  - the label was ink-muted, which E1 verified against PANEL (4.67:1) —
  //    but the hero sits on PAGE, where the same token measures 4.17:1 and
  //    fails AA. ink-soft clears 4.5:1 on every light surface.
  return (
    <dl className={cx('flex flex-col-reverse gap-0.5', className)}>
      <dt
        className={cx(
          'text-xs',
          tone === 'dark' ? 'text-ink-on-dark-soft' : 'text-ink-soft',
        )}
      >
        {label}
      </dt>
      <dd
        className={cx(
          'font-display text-display-xs font-extrabold',
          tone === 'dark' ? 'text-honey' : 'text-ink',
        )}
      >
        {value}
      </dd>
    </dl>
  )
}
