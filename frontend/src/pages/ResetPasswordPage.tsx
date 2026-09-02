import { useState } from 'react'
import { Link, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { ApiError } from '../api/client'
import { useResetPassword } from '../api/hooks'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { useLocale } from '../i18n/useLocale'

/**
 * /reset-password/:token — the emailed link's landing page.
 *
 * The token stays in the URL and is only ever POSTed; the page never stores
 * it anywhere else. A dead link (spent, expired, invented — the server
 * deliberately does not say which) renders one calm message with the way
 * out: ask for a fresh one.
 */
export function ResetPasswordPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { token = '' } = useParams()
  const [password, setPassword] = useState('')
  const reset = useResetPassword()
  const { fieldError } = useFieldErrors(reset.error)

  const tokenRejected =
    reset.error instanceof ApiError && reset.error.code === 'invalid_token'

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    reset.mutate({ token, password })
  }

  return (
    <div className="mx-auto max-w-md px-6 py-14">
      <h1 className="font-display text-display-sm font-extrabold text-ink">
        {t('account:reset.title')}
      </h1>

      {reset.isSuccess ? (
        <div role="status" className="mt-6 rounded-2xl bg-honey/25 p-5 text-sm text-ink-strong">
          {t('account:reset.done')}{' '}
          <Link to={localePath('/login')} className="font-semibold text-brand-ink hover:underline">
            {t('account:signIn')}
          </Link>
        </div>
      ) : tokenRejected ? (
        <div role="alert" className="mt-6 rounded-2xl bg-danger/10 p-5 text-sm text-danger">
          {t('account:reset.invalid')}{' '}
          <Link
            to={localePath('/forgot-password')}
            className="font-semibold text-brand-ink hover:underline"
          >
            {t('account:reset.requestNew')}
          </Link>
        </div>
      ) : (
        <form onSubmit={onSubmit} noValidate className="mt-6 flex flex-col gap-4">
          <Input
            id="reset-password"
            label={t('account:reset.newPassword')}
            type="password"
            autoComplete="new-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={fieldError('password')}
          />
          <Button type="submit" size="lg" disabled={reset.isPending || !password}>
            {reset.isPending ? t('account:working') : t('account:reset.submit')}
          </Button>
        </form>
      )}
    </div>
  )
}
