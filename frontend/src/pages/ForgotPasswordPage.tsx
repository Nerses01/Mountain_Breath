import { useState } from 'react'
import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useForgotPassword } from '../api/hooks'
import { Button } from '../components/ui/Button'
import { Input } from '../components/ui/Input'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { useLocale } from '../i18n/useLocale'

/**
 * /forgot-password — request the reset link.
 *
 * The success copy is deliberately conditional ("if that address is ours…"):
 * the server answers 204 whether or not the email exists, so this page
 * CANNOT know more than that, and pretending otherwise would re-open the
 * enumeration oracle the backend closed.
 */
export function ForgotPasswordPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const [email, setEmail] = useState('')
  const forgot = useForgotPassword()
  const { fieldError, formError } = useFieldErrors(forgot.error)

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    forgot.mutate(email)
  }

  return (
    <div className="mx-auto max-w-md px-6 py-14">
      <h1 className="font-display text-display-sm font-extrabold text-ink">
        {t('account:forgot.title')}
      </h1>
      <p className="mt-2 text-[0.9375rem] text-ink-body">{t('account:forgot.blurb')}</p>

      {forgot.isSuccess ? (
        <p role="status" className="mt-6 rounded-2xl bg-honey/25 p-5 text-sm text-ink-strong">
          {t('account:forgot.sent')}
        </p>
      ) : (
        <form onSubmit={onSubmit} noValidate className="mt-6 flex flex-col gap-4">
          {formError && (
            <p role="alert" className="rounded-xl bg-danger/10 p-4 text-sm text-danger">
              {formError}
            </p>
          )}
          <Input
            id="forgot-email"
            label={t('account:email')}
            type="email"
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            error={fieldError('email')}
          />
          <Button type="submit" size="lg" disabled={forgot.isPending || !email.trim()}>
            {forgot.isPending ? t('account:working') : t('account:forgot.submit')}
          </Button>
        </form>
      )}

      <p className="mt-6 text-sm">
        <Link to={localePath('/login')} className="font-semibold text-brand-ink hover:underline">
          {t('account:forgot.backToSignIn')}
        </Link>
      </p>
    </div>
  )
}
