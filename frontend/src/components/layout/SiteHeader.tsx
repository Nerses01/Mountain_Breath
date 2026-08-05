import { Link, useLocation } from 'react-router'
import { useTranslation } from 'react-i18next'
import { cx } from '../../lib/cx'
import { useLocale } from '../../i18n/useLocale'
import { useCart, useLogout, useMe } from '../../api/hooks'
import { IconButton } from '../ui/IconButton'
import { HeartIcon, SearchIcon, UserIcon } from '../ui/icons'

/**
 * The design's header: brand block, five nav links, icon row, cart pill.
 *
 * Three of the five nav destinations (Our hive, Benefits, Journal) are not
 * built until E9. They render as plain text rather than links, so the header
 * keeps the shape the design gives it without shipping links that navigate
 * to a blank page. E9 turns them into <Link>s by giving them a `to`.
 */
const NAV = [
  { key: 'home', to: '/' },
  { key: 'shop', to: '/shop' },
  { key: 'ourHive', to: undefined },
  { key: 'benefits', to: undefined },
  { key: 'journal', to: undefined },
] as const

export function SiteHeader() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { pathname } = useLocation()
  const me = useMe()
  const cart = useCart(!!me.data)
  const logout = useLogout()

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
                3.6:1 and fails AA (see the token block). */}
            <span className="mt-1 text-2xs uppercase tracking-eyebrow text-ink-muted">
              {t('common:tagline')}
            </span>
          </span>
        </Link>

        <nav aria-label={t('common:nav.home')}>
          <ul className="flex flex-wrap items-center gap-6 font-display text-base font-medium">
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

        <div className="flex items-center gap-3">
          <IconButton label={t('common:actions.search')}>
            <SearchIcon />
          </IconButton>
          <IconButton label={t('common:actions.wishlist')}>
            <HeartIcon />
          </IconButton>

          {/* The design draws no account control at all, since it never
              shows a signed-in state. The app has auth, so one is added
              here (docs/PLAN_ERA_2.md §6, exception 2). */}
          <Link
            to={localePath(me.data ? '/orders' : '/login')}
            aria-label={
              me.data ? t('common:actions.account') : t('common:actions.signIn')
            }
            title={me.data ? me.data.email : t('common:actions.signIn')}
            className="inline-flex size-9.5 items-center justify-center rounded-full border-[1.5px] border-line-strong text-ink-muted transition hover:text-ink"
          >
            <UserIcon />
          </Link>

          {/* Sign-out. The design draws no account menu at all, but a shop
              that can be signed into must be signable out of — the old
              AuthStatus component carried this, and replacing the header
              would otherwise have silently dropped it. It moves into the
              account area proper in E8. */}
          {me.data && (
            <button
              type="button"
              onClick={() => logout.mutate()}
              className="font-display text-sm font-medium text-ink-muted transition hover:text-ink"
            >
              {t('account:signOut')}
            </button>
          )}

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
        </div>
      </div>
    </header>
  )
}
