import { useEffect, useRef, useState } from 'react'
import { Link, useLocation } from 'react-router'
import { useTranslation } from 'react-i18next'
import { cx } from '../../lib/cx'
import { useLocale } from '../../i18n/useLocale'
import { useCart, useLogout, useMe } from '../../api/hooks'
import { IconButton } from '../ui/IconButton'
import { HeartIcon, MenuIcon, SearchIcon, UserIcon, XIcon } from '../ui/icons'
import { SearchOverlay } from './SearchOverlay'

/**
 * The design's header: brand block, five nav links, icon row, cart pill.
 *
 * All five nav destinations are live since E9 gave the last three their
 * pages — the `to: undefined` plain-text state they waited in is gone.
 */
const NAV = [
  { key: 'home', to: '/' },
  { key: 'shop', to: '/shop' },
  { key: 'ourHive', to: '/our-hive' },
  { key: 'benefits', to: '/benefits' },
  { key: 'journal', to: '/journal' },
] as const

export function SiteHeader() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { pathname } = useLocation()
  const me = useMe()
  const cart = useCart(!!me.data)
  const logout = useLogout()

  // E2 moves search out of the catalog body and into an overlay opened from
  // here. The ref is not decoration: closing a modal must put focus back on
  // the control that opened it, or the next Tab restarts from the top of the
  // document and a keyboard user loses their place.
  const [searchOpen, setSearchOpen] = useState(false)
  const searchButtonRef = useRef<HTMLButtonElement>(null)

  const closeSearch = () => {
    setSearchOpen(false)
    searchButtonRef.current?.focus()
  }

  // E10: below md the nav collapses into a disclosure sheet (the plan's 768
  // breakpoint). A DISCLOSURE, not a modal: the page stays interactive
  // behind it, so no focus trap is owed — just aria-expanded/-controls,
  // Escape, and closing on navigation. The route change effect covers every
  // way a link can be activated (click, Enter, middle-click falls out of
  // the page anyway).
  const [menuOpen, setMenuOpen] = useState(false)
  useEffect(() => setMenuOpen(false), [pathname])
  useEffect(() => {
    if (!menuOpen) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMenuOpen(false)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [menuOpen])

  const cartCount = cart.data?.items.reduce((sum, it) => sum + it.qty, 0) ?? 0

  return (
    <header className="bg-linear-to-b from-panel-soft to-panel">
      <div className="mx-auto flex max-w-360 flex-wrap items-center justify-between gap-4 px-6 py-5 lg:px-14">
        <Link to={localePath('/')} className="flex items-center gap-3">
          <span
            aria-hidden
            className="flex size-10 items-center justify-center rounded-md bg-honey font-display text-lg font-extrabold text-ink"
          >
            M
          </span>
          <span className="flex flex-col leading-none">
            <span className="font-display text-lg font-extrabold tracking-wide text-ink">
              {t('common:brand')}
            </span>
            {/* ink-muted, not the design's #a9714b: at 11px that measures
                3.6:1 and fails AA (see the token block).
                Hidden between md and 2xl: that is the squeeze zone where the
                nav and the icon row share the line, and the tagline is the
                widest fixed item in the header once the labels are Armenian.
                Phones keep it (the nav is in the sheet there) and full
                desktop keeps the mock's complete brand block. */}
            <span className="mt-1 text-2xs uppercase tracking-eyebrow text-ink-muted md:max-2xl:hidden">
              {t('common:tagline')}
            </span>
          </span>
        </Link>

        {/* The nav and icon row step down one notch below 2xl — the mock's
            spacing is a 1440px drawing, and the same header must also hold
            five ARMENIAN labels at 1280 without wrapping the icon row onto
            a second line. */}
        <nav aria-label={t('common:nav.home')} className="hidden md:block">
          <ul className="flex flex-wrap items-center gap-x-5 gap-y-1 font-display text-[15px] font-medium 2xl:gap-x-6 2xl:text-base">
            {NAV.map(({ key, to }) => {
              const label = t(`common:nav.${key}`)
              if (!to) {
                return (
                  <li key={key} className="text-ink-muted" aria-disabled>
                    {label}
                  </li>
                )
              }
              const href = localePath(to)
              const isActive = pathname === href
              return (
                <li key={key}>
                  <Link
                    to={href}
                    aria-current={isActive ? 'page' : undefined}
                    className={cx(
                      'transition hover:text-brand-ink',
                      isActive ? 'font-semibold text-brand-ink' : 'text-ink-strong',
                    )}
                  >
                    {label}
                  </Link>
                </li>
              )
            })}
          </ul>
        </nav>

        <div className="flex items-center gap-2.5 2xl:gap-3">
          <IconButton
            ref={searchButtonRef}
            label={t('common:actions.search')}
            aria-expanded={searchOpen}
            onClick={() => setSearchOpen(true)}
          >
            <SearchIcon />
          </IconButton>
          {/* E8: the header heart goes somewhere now. A Link styled as the
              icon button — it is navigation, and middle-click should work.
              Hidden on phones (it lives in the sheet there): five targets
              in a 375px row leaves none of them 44px wide. */}
          <Link
            to={localePath('/wishlist')}
            aria-label={t('common:actions.wishlist')}
            className="hidden size-9.5 items-center justify-center rounded-full border-[1.5px] border-line-strong text-ink-muted transition hover:text-ink sm:inline-flex"
          >
            <HeartIcon />
          </Link>

          {/* No hive-club badge here anymore: the account page shows the
              standing (E8), and the header row is the tightest real estate
              in the app — Armenian labels already fight for the line. */}

          {/* The design draws no account control at all, since it never
              shows a signed-in state. The app has auth, so one is added
              here (docs/PLAN_ERA_2.md §6, exception 2). */}
          <Link
            to={localePath(me.data ? '/account' : '/login')}
            aria-label={
              me.data ? t('common:actions.account') : t('common:actions.signIn')
            }
            title={me.data ? me.data.email : t('common:actions.signIn')}
            className="hidden size-9.5 items-center justify-center rounded-full border-[1.5px] border-line-strong text-ink-muted transition hover:text-ink sm:inline-flex"
          >
            <UserIcon />
          </Link>

          {/* No sign-out here anymore: E8 gave the account area its own (and
              the phone sheet keeps one), so the header carries only what
              every visit needs — which is also what lets the Armenian
              header fit on one line. */}
          <Link
            to={localePath('/cart')}
            className="inline-flex items-center gap-2 rounded-full bg-bark px-5 py-2.5 font-display text-sm font-semibold text-ink-on-dark transition hover:opacity-90"
          >
            {t('common:actions.cart')}
            {cartCount > 0 && (
              <>
                <span aria-hidden>·</span>
                {/* The count is announced as part of the link's text, so a
                    screen reader reads "Cart · 3" in one go. */}
                <span>{cartCount}</span>
              </>
            )}
          </Link>

          <IconButton
            label={t('common:nav.menu')}
            aria-expanded={menuOpen}
            aria-controls="mobile-nav"
            onClick={() => setMenuOpen((v) => !v)}
            className="md:hidden"
          >
            {menuOpen ? <XIcon /> : <MenuIcon />}
          </IconButton>
        </div>
      </div>

      {/* The sheet: the five destinations plus the controls the narrow icon
          row sheds (wishlist, account, sign-out). Rendered in flow rather
          than floated over the page — pushing content down keeps it obvious
          the page underneath is still the page. */}
      {menuOpen && (
        <nav
          id="mobile-nav"
          aria-label={t('common:nav.menu')}
          className="border-t border-line-soft bg-panel-soft md:hidden"
        >
          <ul className="flex flex-col px-6 py-3 font-display text-base font-medium">
            {NAV.map(({ key, to }) => {
              const href = localePath(to)
              return (
                <li key={key}>
                  <Link
                    to={href}
                    aria-current={pathname === href ? 'page' : undefined}
                    className={cx(
                      'block py-3',
                      pathname === href ? 'font-semibold text-brand-ink' : 'text-ink-strong',
                    )}
                  >
                    {t(`common:nav.${key}`)}
                  </Link>
                </li>
              )
            })}
            <li className="mt-1 border-t border-line-soft pt-1">
              <Link to={localePath('/wishlist')} className="block py-3 text-ink-strong">
                {t('common:actions.wishlist')}
              </Link>
            </li>
            <li>
              <Link
                to={localePath(me.data ? '/account' : '/login')}
                className="block py-3 text-ink-strong"
              >
                {me.data ? t('common:actions.account') : t('common:actions.signIn')}
              </Link>
            </li>
            {me.data && (
              <li>
                <button
                  type="button"
                  onClick={() => {
                    logout.mutate()
                    setMenuOpen(false)
                  }}
                  className="block w-full py-3 text-left text-ink-strong"
                >
                  {t('account:signOut')}
                </button>
              </li>
            )}
          </ul>
        </nav>
      )}

      {searchOpen && <SearchOverlay onClose={closeSearch} />}
    </header>
  )
}
