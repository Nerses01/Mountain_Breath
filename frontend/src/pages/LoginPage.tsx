import { useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useLogin, useRegister } from '../api/hooks'
import { Button } from '../components/ui/Button'
import { Checkbox } from '../components/ui/Checkbox'
import { Input } from '../components/ui/Input'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { useLocale } from '../i18n/useLocale'
import { cx } from '../lib/cx'

/**
 * Screen 06 — the two-panel sign-in. Left: the "Hive club / Welcome back"
 * form with show/hide password, keep-me-signed-in, forgot-password, the two
 * provider buttons and the create-account line. Right: the dark panel
 * selling the club (photo slot, headline, the three perk tiles).
 *
 * Sign-in and create-account share the panel: the design draws only
 * sign-in, and its own "New here? Create an account" line is the mode
 * switch, so registering swaps the copy rather than the page.
 *
 * The Google button is an <a>, not a fetch: OAuth is a NAVIGATION — the
 * browser must leave for Google and come back with a session cookie. Apple
 * is decorative-disabled with the truth underneath (decision #5: the paid
 * developer program is not bought), the E6 card-stub pattern again.
 */
export function LoginPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const [params] = useSearchParams()
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [remember, setRemember] = useState(false)

  const navigate = useNavigate()
  const login = useLogin()
  const register = useRegister()
  const active = mode === 'login' ? login : register
  // A cancelled or failed Google round-trip lands back here with a flag —
  // the one thing a redirect flow can carry.
  const oauthFailed = params.get('oauth_error') === '1'

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    const onSuccess = () => navigate(localePath('/'))
    if (mode === 'login') {
      login.mutate({ email, password, remember }, { onSuccess })
    } else {
      register.mutate({ email, password }, { onSuccess })
    }
  }

  const { fieldError, formError } = useFieldErrors(active.error)

  return (
    <div className="mx-auto max-w-360 px-6 py-10 lg:px-14">
      <div className="grid overflow-hidden rounded-2xl lg:grid-cols-2">
        {/* ── The form panel ─────────────────────────────────────────── */}
        <div className="flex flex-col gap-8 bg-card p-8 lg:p-12">
          <div className="flex flex-col gap-2.5">
            <span className="font-display text-2xs font-bold uppercase tracking-eyebrow text-ink-soft">
              {t('account:club.eyebrow')}
            </span>
            <h1 className="font-display text-display-md font-extrabold tracking-tight text-ink">
              {mode === 'login' ? t('account:login.title') : t('account:register.title')}
            </h1>
            <p className="max-w-md text-base text-ink-body">
              {mode === 'login' ? t('account:login.blurb') : t('account:register.blurb')}
            </p>
          </div>

          <form onSubmit={onSubmit} noValidate className="flex w-full max-w-md flex-col gap-4">
            {oauthFailed && mode === 'login' && (
              <p role="alert" className="rounded-xl bg-danger/10 p-4 text-sm text-danger">
                {t('account:login.oauthFailed')}
              </p>
            )}
            {formError && (
              <p role="alert" className="rounded-xl bg-danger/10 p-4 text-sm text-danger">
                {formError}
              </p>
            )}

            <Input
              id="login-email"
              label={t('account:email')}
              type="email"
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              error={fieldError('email')}
            />

            <div className="relative">
              <Input
                id="login-password"
                label={t('account:password')}
                type={showPassword ? 'text' : 'password'}
                autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                error={fieldError('password')}
              />
              {/* The design's "Show" — a real toggle, announced as one. */}
              <button
                type="button"
                onClick={() => setShowPassword((v) => !v)}
                aria-pressed={showPassword}
                className="absolute right-4 top-9.5 text-xs font-semibold text-brand-ink hover:underline"
              >
                {showPassword ? t('account:login.hide') : t('account:login.show')}
              </button>
            </div>

            {mode === 'login' && (
              <div className="flex items-center justify-between gap-3">
                <Checkbox
                  checked={remember}
                  onChange={(e) => setRemember(e.target.checked)}
                  label={t('account:login.remember')}
                />
                <Link
                  to={localePath('/forgot-password')}
                  className="text-sm font-semibold text-brand-ink hover:underline"
                >
                  {t('account:login.forgot')}
                </Link>
              </div>
            )}

            <Button type="submit" size="lg" disabled={active.isPending}>
              {active.isPending
                ? t('account:working')
                : mode === 'login'
                  ? t('account:signIn')
                  : t('account:createAccount')}
            </Button>

            <div
              aria-hidden="true"
              className="flex items-center gap-3.5 text-xs text-ink-muted"
            >
              <span className="h-px flex-1 bg-line" />
              {t('account:login.or')}
              <span className="h-px flex-1 bg-line" />
            </div>

            <div className="flex flex-col gap-3 sm:flex-row">
              {/* A navigation, deliberately — see the doc comment. */}
              <a
                href="/api/v1/auth/oauth/google"
                className={cx(
                  'flex-1 rounded-full border-[1.5px] border-line bg-card px-4 py-3.5',
                  'text-center font-display text-sm font-semibold text-ink',
                  'transition hover:border-line-strong',
                )}
              >
                {t('account:login.google')}
              </a>
              {/* E10 axe fix: the stub was opacity-50, which halves the
                  TEXT's contrast below AA. Inertness now reads from the
                  dashed border and muted-but-legal ink instead — looking
                  disabled is a style; being unreadable is a violation. */}
              <div
                aria-hidden="true"
                className="flex-1 rounded-full border-[1.5px] border-dashed border-line bg-card px-4 py-3.5 text-center font-display text-sm font-semibold text-ink-muted"
              >
                {t('account:login.apple')}
              </div>
            </div>
            <p className="text-xs text-ink-muted">{t('account:login.appleNote')}</p>

            <p className="pt-1 text-sm text-ink-body">
              {mode === 'login' ? (
                <>
                  {t('account:login.newHere')}{' '}
                  <button
                    type="button"
                    onClick={() => setMode('register')}
                    className="font-semibold text-brand-ink hover:underline"
                  >
                    {t('account:createAccount')}
                  </button>{' '}
                  {t('account:login.firstOrderFree')}
                </>
              ) : (
                <>
                  {t('account:register.haveAccount')}{' '}
                  <button
                    type="button"
                    onClick={() => setMode('login')}
                    className="font-semibold text-brand-ink hover:underline"
                  >
                    {t('account:signIn')}
                  </button>
                </>
              )}
            </p>
          </form>
        </div>

        {/* ── The club panel ─────────────────────────────────────────── */}
        <div className="flex flex-col justify-center gap-6 bg-bark p-8 lg:p-12">
          <div
            aria-hidden="true"
            className="flex h-52 items-center justify-center rounded-2xl bg-[repeating-linear-gradient(135deg,rgba(246,194,68,0.28)_0_11px,rgba(246,194,68,0.12)_11px_22px)] text-center font-mono text-2xs uppercase tracking-eyebrow text-honey lg:h-64"
          >
            {t('account:club.imageSlot')}
          </div>
          <p className="font-display text-xl font-bold leading-snug text-ink-on-dark lg:text-2xl">
            {t('account:club.headline')}
          </p>
          <p className="text-[0.9375rem] leading-relaxed text-ink-on-dark-body">
            {t('account:club.blurb')}
          </p>
          <div className="flex flex-col gap-3 sm:flex-row">
            {(['discount', 'pick', 'delivery'] as const).map((key) => (
              <div key={key} className="flex-1 rounded-2xl bg-card/10 p-4.5">
                <div className="font-display text-2xl font-extrabold text-honey">
                  {t(`account:club.tiles.${key}.value`)}
                </div>
                <div className="text-xs text-ink-on-dark-body">
                  {t(`account:club.tiles.${key}.label`)}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}
