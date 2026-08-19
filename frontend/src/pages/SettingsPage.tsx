import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { api, ApiError } from '../api/client'
import {
  useChangePassword,
  useDeleteAccount,
  useMe,
  useNewsletterToggle,
  useNotifications,
  useSetOrderUpdates,
  useUpdateProfile,
} from '../api/hooks'
import type { User } from '../api/types'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { Switch } from '../components/ui/Switch'
import { cx } from '../lib/cx'
import { useCurrency } from '../lib/useCurrency'
import { LOCALES, LOCALE_META } from '../i18n/locales'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { useLocale } from '../i18n/useLocale'

/**
 * /account/settings — canvas 10, in full (A5): profile + password,
 * language & currency, notifications, and the delete-account row.
 *
 * The honesty ledger for this screen:
 *  - profile and password are REAL (PATCH /account/profile,
 *    POST /account/password — other sessions revoked);
 *  - order updates is a real persisted toggle (F2's mailer reads it);
 *  - harvest notes drives the actual newsletter subscription, double
 *    opt-in included — which is why it has a THIRD state (pending);
 *  - wishlist alerts and SMS are disabled with the truth stated
 *    (decision #87 — their columns arrive with their senders);
 *  - delete-account is REAL (F2, decision #97): password-confirmed,
 *    armed like the address book's remove, orders stay in the books —
 *    and the data download next to it is the privacy page's "we will
 *    show you what we store", as one JSON file.
 */
export function SettingsPage() {
  const { t } = useTranslation()
  const me = useMe()

  if (!me.data) return null
  const user = me.data

  return (
    <>
      <div className="flex flex-col gap-1.5">
        <h1 className="font-display text-display-md font-extrabold text-ink">
          {t('account:nav.settings')}
        </h1>
        <p className="text-[0.9375rem] text-ink-soft">{t('account:settingsScreen.subtitle')}</p>
      </div>

      <div className="mt-6 flex flex-col gap-5">
        <ProfilePanel user={user} />

        <div className="grid items-start gap-5 xl:grid-cols-2">
          <LanguageCurrencyPanel />
          <NotificationsPanel user={user} />
        </div>

        <DeleteAccountRow />
      </div>
    </>
  )
}

/** Profile: the read-only boxes until Edit, then the form; the password
 *  row opens its own form. One panel, three modes. */
function ProfilePanel({ user }: { user: User }) {
  const { t } = useTranslation()
  const update = useUpdateProfile()
  const changePassword = useChangePassword()

  type Mode = 'view' | 'edit' | 'password'
  const [mode, setMode] = useState<Mode>('view')
  const [fullName, setFullName] = useState(user.full_name)
  const [phone, setPhone] = useState(user.phone)
  const [current, setCurrent] = useState('')
  const [next, setNext] = useState('')
  const [passwordChanged, setPasswordChanged] = useState(false)

  const profileErrors = useFieldErrors(update.error)
  const passwordErrors = useFieldErrors(changePassword.error)

  const openEdit = () => {
    update.reset()
    setFullName(user.full_name)
    setPhone(user.phone)
    setMode('edit')
  }
  const openPassword = () => {
    changePassword.reset()
    setCurrent('')
    setNext('')
    setPasswordChanged(false)
    setMode('password')
  }

  const submitProfile = (e: React.FormEvent) => {
    e.preventDefault()
    update.mutate({ full_name: fullName, phone }, { onSuccess: () => setMode('view') })
  }
  const submitPassword = (e: React.FormEvent) => {
    e.preventDefault()
    changePassword.mutate(
      { current_password: current, new_password: next },
      {
        onSuccess: () => {
          setMode('view')
          setPasswordChanged(true)
        },
      },
    )
  }

  return (
    <section className="flex flex-col gap-4 rounded-3xl bg-card p-6 sm:p-7">
      <div className="flex items-center justify-between">
        <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
          {t('account:profile.title')}
        </h2>
        {mode === 'view' && (
          <button
            type="button"
            onClick={openEdit}
            className="font-display text-sm font-semibold text-brand-ink hover:underline"
          >
            {t('account:settingsScreen.edit')}
          </button>
        )}
      </div>

      {/* The reset flow's sibling promise, made visible: a changed password
          signs other devices out — say so, or nobody learns it happened. */}
      {passwordChanged && (
        <p role="status" className="rounded-xl bg-honey/25 px-4 py-3 text-sm font-semibold text-ink">
          {t('account:settingsScreen.passwordChanged')}
        </p>
      )}

      {mode !== 'password' && (
        <form onSubmit={submitProfile} noValidate className="grid gap-3.5 sm:grid-cols-2">
          {profileErrors.formError && (
            <p role="alert" className="rounded-xl bg-danger/10 p-3 text-sm text-danger sm:col-span-2">
              {profileErrors.formError}
            </p>
          )}
          {mode === 'edit' ? (
            <>
              <Input
                id="set-name"
                label={t('account:settingsScreen.fullName')}
                autoComplete="name"
                value={fullName}
                onChange={(e) => setFullName(e.target.value)}
                error={profileErrors.fieldError('full_name')}
              />
              <ReadOnlyField label={t('account:email')} value={user.email} />
              <Input
                id="set-phone"
                label={t('account:settingsScreen.phone')}
                type="tel"
                autoComplete="tel"
                value={phone}
                onChange={(e) => setPhone(e.target.value)}
                error={profileErrors.fieldError('phone')}
              />
              <div className="flex items-end gap-3 pb-1">
                <Button type="submit" disabled={update.isPending}>
                  {update.isPending ? t('account:working') : t('account:settingsScreen.save')}
                </Button>
                <button
                  type="button"
                  onClick={() => setMode('view')}
                  className="text-sm font-semibold text-ink-muted hover:text-ink"
                >
                  {t('account:addresses.cancel')}
                </button>
              </div>
            </>
          ) : (
            <>
              <ReadOnlyField
                label={t('account:settingsScreen.fullName')}
                value={user.full_name || t('account:settingsScreen.notSet')}
                muted={!user.full_name}
              />
              <ReadOnlyField label={t('account:email')} value={user.email} />
              <ReadOnlyField
                label={t('account:settingsScreen.phone')}
                value={user.phone || t('account:settingsScreen.notSet')}
                muted={!user.phone}
              />
              <div className="flex flex-col gap-1.5">
                <span className="text-[0.8125rem] font-semibold text-ink-soft">
                  {t('account:password')}
                </span>
                <span className="flex items-center justify-between rounded-xl border-[1.5px] border-line bg-panel px-4 py-3 text-[0.9375rem] text-ink-muted">
                  ••••••••••
                  <button
                    type="button"
                    onClick={openPassword}
                    className="font-display text-sm font-semibold text-brand-ink hover:underline"
                  >
                    {t('account:settingsScreen.change')}
                  </button>
                </span>
              </div>
            </>
          )}
        </form>
      )}

      {mode === 'password' && (
        <form onSubmit={submitPassword} noValidate className="grid gap-3.5 sm:grid-cols-2">
          {passwordErrors.formError && (
            <p role="alert" className="rounded-xl bg-danger/10 p-3 text-sm text-danger sm:col-span-2">
              {passwordErrors.formError}
            </p>
          )}
          <Input
            id="set-current-password"
            label={t('account:settingsScreen.currentPassword')}
            type="password"
            autoComplete="current-password"
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            error={passwordErrors.fieldError('current_password')}
          />
          <Input
            id="set-new-password"
            label={t('account:settingsScreen.newPassword')}
            type="password"
            autoComplete="new-password"
            value={next}
            onChange={(e) => setNext(e.target.value)}
            error={passwordErrors.fieldError('new_password')}
          />
          <div className="flex items-center gap-3 sm:col-span-2">
            <Button type="submit" disabled={changePassword.isPending}>
              {changePassword.isPending
                ? t('account:working')
                : t('account:settingsScreen.savePassword')}
            </Button>
            <button
              type="button"
              onClick={() => setMode('view')}
              className="text-sm font-semibold text-ink-muted hover:text-ink"
            >
              {t('account:addresses.cancel')}
            </button>
          </div>
        </form>
      )}
    </section>
  )
}

function ReadOnlyField({
  label,
  value,
  muted = false,
}: {
  label: string
  value: string
  muted?: boolean
}) {
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[0.8125rem] font-semibold text-ink-soft">{label}</span>
      <span
        className={cx(
          'truncate rounded-xl border-[1.5px] border-line bg-panel px-4 py-3 text-[0.9375rem]',
          muted ? 'text-ink-muted' : 'text-ink',
        )}
      >
        {value}
      </span>
    </div>
  )
}

/** Language segments navigate (a language is part of what a page IS);
 *  currency segments set the client-side lens (decision #89). */
function LanguageCurrencyPanel() {
  const { t } = useTranslation()
  const { locale, hrefFor } = useLocale()
  const { currency, setCurrency, display, setDisplay } = useCurrency()

  const currencyMode: 'dual' | 'USD' | 'AMD' = display === 'dual' ? 'dual' : currency

  return (
    <section className="flex flex-col gap-4 rounded-3xl bg-card p-6 sm:p-7">
      <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
        {t('account:settingsScreen.langCurrencyTitle')}
      </h2>

      <div className="flex flex-col gap-2">
        <span className="text-[0.8125rem] font-semibold text-ink-soft">
          {t('account:settingsScreen.siteLanguage')}
        </span>
        <div className="flex gap-2.5">
          {LOCALES.map((code) => (
            // LINKS, not buttons: switching language navigates to the same
            // page under another prefix — the URL is the locale's source
            // of truth (decision #17).
            <Link
              key={code}
              to={hrefFor(code)}
              aria-current={code === locale ? 'true' : undefined}
              className={cx(
                'flex-1 rounded-xl border py-3 text-center font-display text-sm transition',
                code === locale
                  ? 'border-2 border-brand-ink bg-panel font-bold text-ink'
                  : 'border-[1.5px] border-line font-semibold text-ink-strong hover:border-line-strong',
              )}
            >
              {LOCALE_META[code].nativeName}
            </Link>
          ))}
        </div>
      </div>

      <div className="flex flex-col gap-2">
        <span className="text-[0.8125rem] font-semibold text-ink-soft">
          {t('account:settingsScreen.showPrices')}
        </span>
        <div className="flex gap-2.5" role="group" aria-label={t('account:settingsScreen.showPrices')}>
          <CurrencySegment
            label={t('account:settingsScreen.dualPrices')}
            selected={currencyMode === 'dual'}
            onClick={() => setDisplay('dual')}
          />
          <CurrencySegment
            label="USD"
            selected={currencyMode === 'USD'}
            onClick={() => {
              setCurrency('USD')
              setDisplay('single')
            }}
          />
          <CurrencySegment
            label="AMD"
            selected={currencyMode === 'AMD'}
            onClick={() => {
              setCurrency('AMD')
              setDisplay('single')
            }}
          />
        </div>
      </div>
    </section>
  )
}

function CurrencySegment({
  label,
  selected,
  onClick,
}: {
  label: string
  selected: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      onClick={onClick}
      className={cx(
        'flex-1 rounded-xl border py-3 text-center font-display text-sm transition',
        selected
          ? 'border-2 border-brand-ink bg-panel font-bold text-ink'
          : 'border-[1.5px] border-line font-semibold text-ink-strong hover:border-line-strong',
      )}
    >
      {label}
    </button>
  )
}

/** The four canvas toggles, honestly (decision #87): two real, two inert
 *  with the truth beside them. */
function NotificationsPanel({ user }: { user: User }) {
  const { t } = useTranslation()
  const notifications = useNotifications(true)
  const setOrderUpdates = useSetOrderUpdates()
  const newsletter = useNewsletterToggle()

  const prefs = notifications.data
  const newsletterState = prefs?.newsletter ?? 'none'

  return (
    <section className="flex flex-col gap-4 rounded-3xl bg-card p-6 sm:p-7">
      <h2 className="font-display text-sm font-bold uppercase tracking-label text-ink">
        {t('account:settingsScreen.notifTitle')}
      </h2>

      <Switch
        checked={prefs?.order_updates ?? true}
        disabled={notifications.isPending || setOrderUpdates.isPending}
        onChange={(on) => setOrderUpdates.mutate(on)}
        label={t('account:settingsScreen.orderUpdates')}
        description={t('account:settingsScreen.orderUpdatesDesc')}
      />

      <div className="flex flex-col gap-1.5">
        <Switch
          checked={newsletterState !== 'none'}
          disabled={
            notifications.isPending || newsletter.subscribe.isPending || newsletter.unsubscribe.isPending
          }
          onChange={(on) => {
            if (on) newsletter.subscribe.mutate(user.email)
            else newsletter.unsubscribe.mutate()
          }}
          label={t('account:settingsScreen.harvestNotes')}
          description={t('account:settingsScreen.harvestNotesDesc')}
        />
        {/* Double opt-in's middle state, shown instead of pretended away:
            the toggle is on, the inbox has the last word. */}
        {newsletterState === 'pending' && (
          <p role="status" className="text-[0.8125rem] font-semibold text-brand-ink">
            {t('account:settingsScreen.harvestPending')}
          </p>
        )}
      </div>

      <div className="border-t border-line-soft" />

      {/* The two channels with no sender: visible, inert, explained. */}
      <Switch
        checked={false}
        disabled
        onChange={() => {}}
        label={t('account:settingsScreen.wishlistAlerts')}
        description={t('account:settingsScreen.wishlistAlertsDesc')}
      />
      <Switch
        checked={false}
        disabled
        onChange={() => {}}
        label={t('account:settingsScreen.smsTitle')}
        description={t('account:settingsScreen.smsDesc')}
      />
      <p className="rounded-xl bg-panel px-4 py-3 text-[0.8125rem] text-ink-muted">
        {t('account:settingsScreen.stubNote')}
      </p>
    </section>
  )
}

/** Canvas 10's danger row — F2 owns the endpoint; until it exists the
 *  button is disabled with the policy stated, not a dead click. */
/** F2 (decision #97): the stub grew its endpoint. Collapsed, the row is
 *  the canvas's "Delete account / Delete…"; expanded, it asks for the
 *  current password (Google-only accounts have none — the hint says so),
 *  offers the data download, and the button itself arms for 3 s before
 *  it fires — the address book's confirm pattern at higher stakes. */
function DeleteAccountRow() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const navigate = useNavigate()
  const del = useDeleteAccount()

  const [open, setOpen] = useState(false)
  const [password, setPassword] = useState('')
  const [downloading, setDownloading] = useState(false)
  const [arming, setArming] = useState(false)
  const disarm = useRef<ReturnType<typeof setTimeout>>(undefined)
  useEffect(() => () => clearTimeout(disarm.current), [])

  const download = async () => {
    setDownloading(true)
    try {
      const data = await api.accountData()
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = 'mountain-breath-data.json'
      a.click()
      URL.revokeObjectURL(url)
    } finally {
      setDownloading(false)
    }
  }

  const deleteClick = () => {
    if (arming) {
      clearTimeout(disarm.current)
      setArming(false)
      del.mutate(password, {
        onSuccess: () => navigate(localePath('/')),
      })
      return
    }
    setArming(true)
    disarm.current = setTimeout(() => setArming(false), 3000)
  }

  const { fieldError } = useFieldErrors(del.error)
  const lastAdmin = del.error instanceof ApiError && del.error.code === 'last_admin'

  return (
    <section className="rounded-3xl border-[1.5px] border-line bg-panel px-6 py-5">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex min-w-0 flex-col gap-1">
          <h2 className="font-display text-[0.9375rem] font-bold text-ink">
            {t('account:settingsScreen.deleteTitle')}
          </h2>
          <p className="text-[0.8125rem] text-ink-muted">
            {t('account:settingsScreen.deleteBlurb')}{' '}
            <Link to={localePath('/privacy')} className="font-semibold text-brand-ink hover:underline">
              {t('account:settingsScreen.privacyLink')}
            </Link>
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            type="button"
            disabled={downloading}
            onClick={download}
            className="rounded-full border-[1.5px] border-line px-5 py-2.5 font-display text-sm font-semibold text-ink hover:border-ink disabled:opacity-50"
          >
            {downloading
              ? t('account:working')
              : t('account:settingsScreen.downloadData')}
          </button>
          {!open && (
            <button
              type="button"
              onClick={() => setOpen(true)}
              className="rounded-full border-[1.5px] border-danger px-5 py-2.5 font-display text-sm font-semibold text-danger hover:bg-danger hover:text-white"
            >
              {t('account:settingsScreen.deleteButton')}
            </button>
          )}
        </div>
      </div>

      {open && (
        <div className="mt-4 flex flex-col gap-3 border-t border-line pt-4 sm:max-w-sm">
          <Input
            label={t('account:settingsScreen.currentPassword')}
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={fieldError('current_password')}
          />
          <p className="text-[0.8125rem] text-ink-muted">
            {t('account:settingsScreen.deletePasswordHint')}
          </p>
          {lastAdmin && (
            <p role="alert" className="text-[0.8125rem] font-semibold text-danger">
              {t('account:settingsScreen.deleteLastAdmin')}
            </p>
          )}
          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={del.isPending}
              onClick={deleteClick}
              className={cx(
                'rounded-full border-[1.5px] border-danger px-5 py-2.5 font-display text-sm font-semibold disabled:opacity-50',
                arming ? 'bg-danger text-white' : 'text-danger hover:bg-danger hover:text-white',
              )}
            >
              {arming
                ? t('account:settingsScreen.deleteConfirm')
                : t('account:settingsScreen.deleteButton')}
            </button>
            <button
              type="button"
              onClick={() => {
                setOpen(false)
                setPassword('')
              }}
              className="font-display text-sm font-semibold text-ink-soft hover:underline"
            >
              {t('account:settingsScreen.cancel')}
            </button>
          </div>
        </div>
      )}
    </section>
  )
}
