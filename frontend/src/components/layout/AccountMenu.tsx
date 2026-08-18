import { useEffect, useRef, useState } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useLogout } from '../../api/hooks'
import type { User } from '../../api/types'
import { useLocale } from '../../i18n/useLocale'
import { ChevronDownIcon } from '../ui/icons'

/**
 * A1: the header's signed-in control — the canvas's dark pill (avatar
 * initial, name, ▾) opening a dropdown to the four account screens and
 * sign-out. This is the header's "account menu open state" the canvas draws
 * at the top of every account screen.
 *
 * Pattern: MENU BUTTON (WAI-ARIA APG "menu button"), and deliberately not a
 * third ad-hoc popup. The app now has three owned popups, each matching its
 * job: PillSelect is a COMBOBOX (picks a value), the mobile nav is a
 * DISCLOSURE (shows a region), this is a MENU (a list of commands/links).
 * The roles dictate the keyboard contract: menus own real focus — arrows
 * move DOM focus between items (roving tabindex), not an aria-activedescendant
 * highlight — Escape closes and returns focus to the button, Tab closes and
 * moves on. Screen readers announce "menu, 5 items" and expect exactly that
 * behaviour; giving the right role and the wrong keys is worse than divs.
 */
const ITEMS = [
  { key: 'orders', to: '/account/orders' },
  { key: 'wishlist', to: '/account/wishlist' },
  { key: 'addresses', to: '/account/addresses' },
  { key: 'settings', to: '/account/settings' },
] as const

export function AccountMenu({ user }: { user: User }) {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const logout = useLogout()

  const [open, setOpen] = useState(false)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const rootRef = useRef<HTMLDivElement>(null)
  // One ref holding all item nodes (4 links + sign-out) for roving focus.
  const itemsRef = useRef<(HTMLElement | null)[]>([])

  // Until A5 adds users.full_name, the pill shows the email's local part —
  // same rule as the rail, one identity everywhere.
  const displayName = user.email.split('@')[0]

  const close = (refocus: boolean) => {
    setOpen(false)
    if (refocus) buttonRef.current?.focus()
  }

  const openAndFocus = (which: 'first' | 'last') => {
    setOpen(true)
    // The menu is not in the DOM until after this render commits, so both
    // the focus move AND the index math wait a frame: before the commit,
    // itemsRef is still empty, so "last" cannot be computed yet. (The C++
    // instinct — "it's right there, call it" — trips on React's deferred
    // rendering: state now, DOM later.)
    requestAnimationFrame(() => {
      const items = itemsRef.current
      items[which === 'first' ? 0 : items.length - 1]?.focus()
    })
  }

  // Light dismiss: a click anywhere outside closes WITHOUT stealing the
  // click (the user meant the thing they clicked, not "close the menu").
  useEffect(() => {
    if (!open) return
    const onPointerDown = (e: PointerEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('pointerdown', onPointerDown)
    return () => document.removeEventListener('pointerdown', onPointerDown)
  }, [open])

  const onButtonKeyDown = (e: React.KeyboardEvent) => {
    // APG: Down/Enter/Space open and focus the FIRST item, Up opens and
    // focuses the LAST — a keyboard user landing at the far end on purpose.
    if (e.key === 'ArrowDown' || e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      openAndFocus('first')
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      openAndFocus('last')
    }
  }

  const onMenuKeyDown = (e: React.KeyboardEvent) => {
    const items = itemsRef.current
    const current = items.findIndex((el) => el === document.activeElement)
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      items[(current + 1) % items.length]?.focus()
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      items[(current - 1 + items.length) % items.length]?.focus()
    } else if (e.key === 'Home') {
      e.preventDefault()
      items[0]?.focus()
    } else if (e.key === 'End') {
      e.preventDefault()
      items[items.length - 1]?.focus()
    } else if (e.key === 'Escape') {
      e.preventDefault()
      close(true)
    } else if (e.key === 'Tab') {
      // Tab is NOT trapped in a menu — it closes and lets focus move on.
      setOpen(false)
    }
  }

  return (
    <div ref={rootRef} className="relative hidden sm:block">
      <button
        ref={buttonRef}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls="account-menu"
        title={user.email}
        onClick={() => (open ? close(true) : openAndFocus('first'))}
        onKeyDown={onButtonKeyDown}
        className="flex items-center gap-2.5 rounded-full bg-bark py-1.5 pl-1.5 pr-3.5 text-ink-on-dark transition hover:opacity-90"
      >
        <span
          aria-hidden
          className="flex size-7.5 items-center justify-center rounded-full bg-honey font-display text-xs font-extrabold text-ink"
        >
          {displayName[0]?.toUpperCase()}
        </span>
        <span className="max-w-28 truncate font-display text-sm font-semibold">
          {displayName}
        </span>
        <ChevronDownIcon size={14} className={open ? 'rotate-180' : undefined} />
      </button>

      {open && (
        <div
          id="account-menu"
          role="menu"
          aria-label={t('common:actions.account')}
          onKeyDown={onMenuKeyDown}
          className="absolute right-0 top-full z-20 mt-2 flex w-56 flex-col gap-0.5 rounded-2xl border border-line-soft bg-card p-2 shadow-screen"
        >
          {ITEMS.map(({ key, to }, i) => (
            <Link
              key={key}
              ref={(el) => {
                itemsRef.current[i] = el
              }}
              role="menuitem"
              // Roving tabindex: items are reachable by arrows, not by Tab —
              // the whole menu is ONE tab stop (the button).
              tabIndex={-1}
              to={localePath(to)}
              onClick={() => setOpen(false)}
              className="rounded-xl px-3.5 py-2.5 font-display text-sm font-semibold text-ink-strong transition hover:bg-panel focus:bg-panel"
            >
              {t(`account:nav.${key}`)}
            </Link>
          ))}
          <div role="presentation" className="mx-2 my-1 border-t border-line-soft" />
          <button
            ref={(el) => {
              itemsRef.current[ITEMS.length] = el
            }}
            type="button"
            role="menuitem"
            tabIndex={-1}
            onClick={() => {
              logout.mutate()
              close(false)
            }}
            className="rounded-xl px-3.5 py-2.5 text-left font-display text-sm font-semibold text-ink-muted transition hover:bg-panel hover:text-ink focus:bg-panel"
          >
            {t('account:signOut')}
          </button>
        </div>
      )}
    </div>
  )
}
