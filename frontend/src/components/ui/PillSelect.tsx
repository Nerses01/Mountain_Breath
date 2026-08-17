import { useEffect, useId, useRef, useState } from 'react'
import { cx } from '../../lib/cx'
import { CheckIcon, ChevronDownIcon } from './icons'

export type PillSelectOption<V extends string> = { value: V; label: string }

/**
 * The design's pill dropdown ("Sort: Most loved ▾") with a popup that
 * belongs to this design instead of to the operating system.
 *
 * WHY NOT <select>: the closed box of a native select can be styled, but
 * the OPEN list is drawn by the OS and takes no CSS at all — the blue
 * Windows menu in the middle of a honey-and-bark page. The trade for
 * owning the pixels is owning the behaviour, so this implements the
 * WAI-ARIA "select-only combobox" pattern by hand:
 *
 *  - The trigger is a real <button> carrying role="combobox",
 *    aria-expanded and aria-controls. DOM focus STAYS on the button the
 *    whole time; the option the keyboard is "on" is only pointed at via
 *    aria-activedescendant. That is the pattern's core trick — one tab
 *    stop, no focus juggling, and closing needs no focus restoration
 *    because focus never left.
 *  - Closed: Enter, Space or either arrow opens, with the highlight on
 *    the CURRENT value, not the first row.
 *  - Open: arrows move the highlight (clamped, no wrap — a five-item sort
 *    list is not a carousel), Home/End jump, Enter/Space commit, Escape
 *    abandons, Tab commits and moves on (per the pattern: Tab is "I'm
 *    done here", not "cancel").
 *  - A pointer press anywhere outside closes without committing.
 *
 * The canvas draws only the CLOSED pill; the open list is a state it never
 * shows (§6 exception 2), styled here from the same tokens as every other
 * floating panel.
 */
export function PillSelect<V extends string>({
  options,
  value,
  onChange,
  ariaLabel,
  prefix,
  className,
}: {
  options: PillSelectOption<V>[]
  value: V
  onChange: (value: V) => void
  /** The control's accessible name — the design puts no label text outside the pill. */
  ariaLabel: string
  /** Shown on the closed pill before the selected label ("Sort:"). */
  prefix?: string
  className?: string
}) {
  const [open, setOpen] = useState(false)
  const selectedIndex = Math.max(
    0,
    options.findIndex((o) => o.value === value),
  )
  const [activeIndex, setActiveIndex] = useState(selectedIndex)
  const rootRef = useRef<HTMLDivElement>(null)
  const listId = useId()

  const openList = () => {
    setActiveIndex(selectedIndex)
    setOpen(true)
  }

  const commit = (index: number) => {
    setOpen(false)
    const picked = options[index]
    if (picked && picked.value !== value) onChange(picked.value)
  }

  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    return () => document.removeEventListener('pointerdown', onPointerDown)
  }, [open])

  const onKeyDown = (e: React.KeyboardEvent) => {
    if (!open) {
      if (['Enter', ' ', 'ArrowDown', 'ArrowUp'].includes(e.key)) {
        e.preventDefault()
        openList()
      }
      return
    }
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault()
        setActiveIndex((i) => Math.min(i + 1, options.length - 1))
        break
      case 'ArrowUp':
        e.preventDefault()
        setActiveIndex((i) => Math.max(i - 1, 0))
        break
      case 'Home':
        e.preventDefault()
        setActiveIndex(0)
        break
      case 'End':
        e.preventDefault()
        setActiveIndex(options.length - 1)
        break
      case 'Enter':
      case ' ':
        e.preventDefault()
        commit(activeIndex)
        break
      case 'Escape':
        e.preventDefault()
        setOpen(false)
        break
      case 'Tab':
        // No preventDefault: commit AND let focus move on.
        commit(activeIndex)
        break
    }
  }

  const selected = options[selectedIndex]

  return (
    <div ref={rootRef} className={cx('relative', className)}>
      <button
        type="button"
        role="combobox"
        aria-expanded={open}
        aria-haspopup="listbox"
        aria-controls={listId}
        aria-label={ariaLabel}
        aria-activedescendant={open ? `${listId}-${activeIndex}` : undefined}
        onClick={() => (open ? setOpen(false) : openList())}
        onKeyDown={onKeyDown}
        className="inline-flex items-center gap-2 rounded-full border-[1.5px] border-line bg-card px-5 py-2.5 font-display text-sm font-semibold text-ink transition hover:border-line-strong"
      >
        <span>
          {prefix ? `${prefix} ` : ''}
          {selected?.label}
        </span>
        <ChevronDownIcon
          size={16}
          className={cx('shrink-0 transition-transform', open && 'rotate-180')}
        />
      </button>

      {open && (
        <ul
          role="listbox"
          id={listId}
          aria-label={ariaLabel}
          className="absolute right-0 top-full z-20 mt-2 min-w-full whitespace-nowrap rounded-2xl border border-line bg-card p-1.5 shadow-screen"
        >
          {options.map((o, i) => {
            const isSelected = o.value === value
            return (
              <li
                key={o.value}
                id={`${listId}-${i}`}
                role="option"
                aria-selected={isSelected}
                // The rows are pointed at, never focused — so the press must
                // not steal focus from the button (default pointerdown
                // behaviour would focus <body> and drop the keyboard user).
                onPointerDown={(e) => e.preventDefault()}
                onMouseEnter={() => setActiveIndex(i)}
                onClick={() => commit(i)}
                className={cx(
                  'flex cursor-pointer items-center justify-between gap-6 rounded-lg px-3.5 py-2.5 font-display text-sm',
                  i === activeIndex && 'bg-panel',
                  isSelected ? 'font-semibold text-brand-ink' : 'text-ink',
                )}
              >
                {o.label}
                {/* The highlight also travels with the keyboard, so the tick
                    is what marks "current" as distinct from "under the
                    cursor". */}
                {isSelected && <CheckIcon size={14} className="shrink-0" />}
              </li>
            )
          })}
        </ul>
      )}
    </div>
  )
}
