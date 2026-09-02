import { Link } from 'react-router'
import { cx } from '../../lib/cx'

/**
 * The design's recurring section header: a wide-tracked uppercase eyebrow,
 * a display title, and an optional link pushed to the right ("Shop all →").
 *
 * Both muted colours here are the corrected tokens rather than the mock's:
 * the eyebrow at 13px and the action link at 15px would both fail AA in the
 * design's original #a9714b / #e4761f (3.6:1 and 2.7:1).
 */
export function SectionHeading({
  eyebrow,
  title,
  action,
  size = 'md',
  className,
}: {
  eyebrow?: string
  title: string
  action?: { label: string; to: string }
  size?: 'sm' | 'md' | 'lg'
  className?: string
}) {
  const titleSize = {
    sm: 'text-display-xs',
    md: 'text-display-md',
    lg: 'text-display-lg',
  }[size]

  return (
    <div className={cx('flex flex-wrap items-end justify-between gap-4', className)}>
      <div className="flex flex-col gap-2">
        {eyebrow && (
          <span className="font-display text-xs font-bold uppercase tracking-eyebrow text-ink-soft">
            {eyebrow}
          </span>
        )}
        <h2 className={cx('font-display font-extrabold text-ink', titleSize)}>
          {title}
        </h2>
      </div>

      {action && (
        <Link
          to={action.to}
          className="font-display text-sm font-semibold text-brand-ink hover:underline"
        >
          {action.label} →
        </Link>
      )}
    </div>
  )
}
