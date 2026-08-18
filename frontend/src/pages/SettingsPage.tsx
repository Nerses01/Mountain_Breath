import { useTranslation } from 'react-i18next'
import { useMe } from '../api/hooks'

/**
 * /account/settings — A1's interim pane.
 *
 * Canvas screen 10 wants profile editing, password change, language &
 * currency and notification toggles — all of which need A5's backend
 * (users.full_name/phone, POST /account/password, preferences). Shipping
 * decorative controls that do nothing is exactly what this project refuses
 * (the E6/E8 stub rule), so until A5 this pane shows only what is real
 * today: who is signed in and their hive standing — the content the old
 * /account profile card carried, so nothing E8 shipped is lost in the move.
 */
export function SettingsPage() {
  const { t } = useTranslation()
  const me = useMe()

  // The shell renders this pane only when signed in; a null here is just the
  // brief logout transition, and rendering nothing beats a flash of "…".
  if (!me.data) return null
  const user = me.data

  return (
    <>
      <h1 className="font-display text-display-md font-extrabold text-ink">
        {t('account:nav.settings')}
      </h1>

      <section className="mt-7 flex max-w-xl flex-col gap-3 rounded-2xl bg-card p-6">
        <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
          {t('account:profile.title')}
        </h2>
        <p className="text-[0.9375rem] text-ink-strong">{user.email}</p>
        {user.hive.member ? (
          <p className="flex items-center gap-2 text-sm text-ink-body">
            <span className="rounded-full bg-honey px-3 py-1 font-display text-xs font-bold text-ink">
              {t('common:hive.badge')}
            </span>
            {t('account:profile.memberLine', {
              percent: user.hive.member_discount_percent,
            })}
          </p>
        ) : (
          <p className="text-sm text-ink-body">{t('account:profile.firstOrderLine')}</p>
        )}
      </section>
    </>
  )
}
