import { Link, NavLink, Outlet, useLocation } from 'react-router'
import { Trans, useTranslation } from 'react-i18next'
import { useAddresses, useLogout, useMe, useMyOrders, useReorder, useWishlist } from '../../api/hooks'
import type { User } from '../../api/types'
import { cx } from '../../lib/cx'
import { displayName, initials } from '../../lib/displayName'
import { formatMoney } from '../../lib/format'
import { useLocale } from '../../i18n/useLocale'
import {
  GearIcon,
  HeartIcon,
  HomeIcon,
  LogoutIcon,
  OrdersIcon,
} from '../ui/icons'

/**
 * A1 (account canvas 07–10): the shell all four account screens share — the
 * left rail (profile card, nav, log out) and a pane the child route fills.
 *
 * This is a LAYOUT ROUTE: it renders around whatever child route matched,
 * and <Outlet /> below is the hole the child appears in. The C++ analogue is
 * the Template Method pattern — the frame is fixed here, the variable step
 * is supplied by someone else — except the router does the wiring instead of
 * a virtual call: nesting <Route element={<AccountLayout/>}> gives every
 * child this frame without any child knowing about it.
 *
 * The signed-out guard lives here ONCE, replacing the three per-page
 * `signInRequired` blocks E8's pages each carried. Children may therefore
 * assume a signed-in user: the <Outlet /> renders only on the happy path.
 */
export function AccountLayout() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const me = useMe()

  if (me.isPending) {
    return <Shell>{t('common:state.loading')}</Shell>
  }
  if (!me.data) {
    return (
      <Shell>
        <p className="text-ink-body">
          <Trans
            i18nKey="account:shellSignIn"
            components={[
              <span key="0" />,
              <Link
                key="1"
                to={localePath('/login')}
                className="font-semibold text-brand-ink hover:underline"
              />,
            ]}
          />
        </p>
      </Shell>
    )
  }

  return (
    <Shell>
      {/* The canvas's 300px + 1fr frame at 1440. Below lg the rail stacks
          above the pane — a state the desktop-only canvas never draws.
          min-w-0 on the pane: a grid child's implicit min-width is `auto`,
          so one wide order row would otherwise widen the whole column
          instead of scrolling inside it. */}
      <div className="grid items-start gap-6 lg:grid-cols-[300px_1fr] lg:gap-8">
        <AccountRail user={me.data} />
        <div className="min-w-0">
          <Outlet />
        </div>
      </div>
    </Shell>
  )
}

/**
 * The rail: the dark profile card with its two stat tiles, then the nav
 * list with counts, a divider, and Log out as the last row — the canvas's
 * exact order. Sign-out moves here from the old account page: the rail is
 * on every account screen, so the door out is too.
 */
function AccountRail({ user }: { user: User }) {
  const { t } = useTranslation()
  const logout = useLogout()

  // The counts beside the nav labels. All three queries are already cached
  // for any visitor who has touched these screens (TanStack keys are
  // shared app-wide), so the rail costs at most three small reads once.
  // `undefined` while loading renders as no count rather than a spinner —
  // a number appearing beats three tiny loaders.
  const orders = useMyOrders()
  const wishlist = useWishlist(true)
  const addresses = useAddresses(true)

  return (
    <div className="flex flex-col gap-4">
      {/* ── Profile card (canvas: dark, avatar, name, standing, tiles) ── */}
      <section className="flex flex-col gap-4 rounded-3xl bg-bark p-6">
        <div className="flex items-center gap-3.5">
          <span
            aria-hidden
            className="flex size-12 shrink-0 items-center justify-center rounded-full bg-honey font-display text-lg font-extrabold text-ink"
          >
            {initials(user)}
          </span>
          <div className="min-w-0">
            <p className="truncate font-display text-base font-bold text-ink-on-dark" title={user.email}>
              {displayName(user)}
            </p>
            <p className="truncate text-xs text-ink-on-dark-soft">{user.email}</p>
          </div>
        </div>
        <p className="text-sm text-ink-on-dark-body">
          {user.hive.member
            ? t('account:rail.member')
            : t('account:rail.guest')}
        </p>
        <div className="flex gap-2.5">
          <StatTile
            value={orders.data ? String(orders.data.length) : '–'}
            label={t('account:rail.ordersTile')}
          />
          {user.hive.member ? (
            <StatTile
              value={`${user.hive.member_discount_percent}%`}
              label={t('account:rail.memberTile')}
            />
          ) : (
            // The non-member tile is ours to design (the canvas only draws a
            // member): the hive-club welcome perk the shop actually grants.
            <StatTile
              value={t('account:rail.firstFreeValue')}
              label={t('account:rail.firstFreeTile')}
            />
          )}
        </div>
      </section>

      {/* ── Nav (canvas: light card, active row filled, counts) ──────── */}
      <nav
        aria-label={t('common:actions.account')}
        className="flex flex-col gap-1 rounded-3xl bg-card p-3.5"
      >
        <RailLink to="/account/orders" icon={<OrdersIcon size={18} />} label={t('account:nav.orders')} count={orders.data?.length} />
        <RailLink to="/account/wishlist" icon={<HeartIcon size={18} />} label={t('account:nav.wishlist')} count={wishlist.data?.length} />
        <RailLink to="/account/addresses" icon={<HomeIcon size={18} />} label={t('account:nav.addresses')} count={addresses.data?.length} />
        <RailLink to="/account/settings" icon={<GearIcon size={18} />} label={t('account:nav.settings')} />
        <div role="presentation" className="mx-2 my-1.5 border-t border-line-soft" />
        <button
          type="button"
          onClick={() => logout.mutate()}
          className="flex items-center gap-3 rounded-xl px-4 py-3 text-left font-display text-[0.9375rem] font-semibold text-ink-muted transition hover:bg-panel hover:text-ink"
        >
          <span aria-hidden className="w-5 text-ink-faint">
            <LogoutIcon size={18} />
          </span>
          {t('account:signOut')}
        </button>
      </nav>

      <RailReorderCard />
      <RailAlertsCard />
    </div>
  )
}

/**
 * A2: the canvas's honey "Reorder in one tap" card — contextual to the
 * orders screen (each screen gets its own rail promo, per the canvas), and
 * only when there is a delivered order to repeat. It calls the SAME
 * reorder path as the history rows; its own confirmation replaces the
 * button so the rail never needs the page's banner.
 */
function RailReorderCard() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { pathname } = useLocation()
  const orders = useMyOrders()
  const reorder = useReorder()

  if (!pathname.includes('/account/orders')) return null
  const last = orders.data?.find((o) => o.status === 'delivered')
  if (!last) return null

  return (
    <section className="flex flex-col gap-2 rounded-3xl bg-honey p-6">
      <h2 className="font-display text-base font-bold text-ink">
        {t('account:railReorder.title')}
      </h2>
      <p className="text-sm leading-relaxed text-ink-strong">
        {t('account:railReorder.usual', {
          items: last.items.map((it) => it.name).join(' · '),
        })}
      </p>
      {reorder.data ? (
        <p role="status" className="mt-1 text-sm font-semibold text-ink">
          {t('account:railReorder.done')}{' '}
          <Link to={localePath('/cart')} className="underline">
            {t('account:ordersScreen.viewCart')}
          </Link>
        </p>
      ) : (
        <button
          type="button"
          onClick={() => reorder.mutate(last.id)}
          disabled={reorder.isPending}
          className="mt-1 rounded-full bg-bark px-5 py-3 text-center font-display text-sm font-bold text-ink-on-dark transition hover:opacity-90 disabled:opacity-50"
        >
          {t('account:railReorder.cta', {
            total: formatMoney(last.total_minor, last.currency),
          })}
        </button>
      )}
    </section>
  )
}

function StatTile({ value, label }: { value: string; label: string }) {
  return (
    <div className="flex-1 rounded-2xl bg-ink-on-dark/8 px-3.5 py-3">
      <p className="font-display text-xl font-extrabold text-honey">{value}</p>
      <p className="text-xs text-ink-on-dark-soft">{label}</p>
    </div>
  )
}

function RailLink({
  to,
  icon,
  label,
  count,
}: {
  to: string
  icon: React.ReactNode
  label: string
  count?: number
}) {
  const { localePath } = useLocale()
  return (
    // NavLink instead of Link: it computes "am I the current route?" itself
    // (prefix match, so /account/orders/42 still lights "My orders") and
    // sets aria-current="page" — the same fact styled and announced from
    // one source, where the header's manual pathname === href would treat
    // the order detail as nowhere.
    <NavLink
      to={localePath(to)}
      className={({ isActive }) =>
        cx(
          'flex items-center gap-3 rounded-xl px-4 py-3 font-display text-[0.9375rem] transition',
          // The canvas fills the active row #E4761F with white text — that
          // pair measures ~2.9:1 and fails AA, so the active fill is
          // --color-brand-ink, the same substitution every button already
          // makes (accessibility exception to rule #16).
          isActive
            ? 'bg-brand-ink font-bold text-ink-on-dark'
            : 'font-semibold text-ink-strong hover:bg-panel',
        )
      }
    >
      {({ isActive }) => (
        <>
          <span aria-hidden className={cx('w-5', isActive ? '' : 'text-ink-faint')}>
            {icon}
          </span>
          {label}
          {count !== undefined && (
            <span
              className={cx(
                'ml-auto text-[0.8125rem]',
                isActive
                  ? 'rounded-full bg-card px-2.5 py-0.5 text-xs font-extrabold text-brand-ink'
                  : 'font-semibold text-ink-muted',
              )}
            >
              {count}
            </span>
          )}
        </>
      )}
    </NavLink>
  )
}

/**
 * A3: canvas 08's dark "Price-drop alerts" card — as the E6/E8 HONEST STUB
 * (decision #87): no sender exists yet, so there is no live toggle to lie
 * with. The card keeps the canvas's promise visible and states plainly
 * that it is not wired; its real toggle arrives with the wishlist mailer.
 */
function RailAlertsCard() {
  const { t } = useTranslation()
  const { pathname } = useLocation()

  if (!pathname.includes('/account/wishlist')) return null

  return (
    <section className="flex flex-col gap-2.5 rounded-3xl bg-bark p-6">
      <h2 className="font-display text-base font-bold text-ink-on-dark">
        {t('account:railAlerts.title')}
      </h2>
      <p className="text-sm leading-relaxed text-ink-on-dark-body">
        {t('account:railAlerts.blurb')}
      </p>
      <p className="mt-1 rounded-xl bg-ink-on-dark/8 px-4 py-3 text-[0.8125rem] text-ink-on-dark-soft">
        {t('account:railAlerts.comingSoon')}
      </p>
    </section>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-360 px-6 py-10 lg:px-14">{children}</div>
}
