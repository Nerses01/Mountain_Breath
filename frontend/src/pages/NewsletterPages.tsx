import { Link, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { UseMutationResult } from '@tanstack/react-query'
import { useConfirmNewsletter, useUnsubscribeNewsletter } from '../api/hooks'
import { Button } from '../components/ui/Button'
import { useLocale } from '../i18n/useLocale'

/**
 * The emailed links' landing pages (E9). Both REQUIRE a button press —
 * nothing fires on mount, and that is the security half of double opt-in's
 * design here: mail scanners prefetch links, and some of them execute
 * JavaScript, so even an auto-POST-on-load page could be "clicked" by a
 * robot. A form submission is the one thing scanners do not do. The human
 * pressing the button IS the consent.
 */
export function NewsletterConfirmPage() {
  const confirm = useConfirmNewsletter()
  const { t } = useTranslation()
  return (
    <TokenActionPage
      mutation={confirm}
      title={t('journal:newsletter.confirmTitle')}
      blurb={t('journal:newsletter.confirmBlurb')}
      action={t('journal:newsletter.confirmAction')}
      done={t('journal:newsletter.confirmDone')}
    />
  )
}

export function NewsletterUnsubscribePage() {
  const unsubscribe = useUnsubscribeNewsletter()
  const { t } = useTranslation()
  return (
    <TokenActionPage
      mutation={unsubscribe}
      title={t('journal:newsletter.unsubscribeTitle')}
      blurb={t('journal:newsletter.unsubscribeBlurb')}
      action={t('journal:newsletter.unsubscribeAction')}
      done={t('journal:newsletter.unsubscribeDone')}
    />
  )
}

function TokenActionPage({
  mutation,
  title,
  blurb,
  action,
  done,
}: {
  mutation: UseMutationResult<void, Error, string>
  title: string
  blurb: string
  action: string
  done: string
}) {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const { token = '' } = useParams()

  return (
    <div className="mx-auto max-w-md px-6 py-14">
      <h1 className="font-display text-display-sm font-extrabold text-ink">{title}</h1>

      {mutation.isSuccess ? (
        <p role="status" className="mt-6 rounded-2xl bg-honey/25 p-5 text-sm text-ink-strong">
          {done}
        </p>
      ) : mutation.isError ? (
        <p role="alert" className="mt-6 rounded-2xl bg-danger/10 p-5 text-sm text-danger">
          {t('journal:newsletter.invalid')}
        </p>
      ) : (
        <>
          <p className="mt-2 text-[0.9375rem] text-ink-body">{blurb}</p>
          <div className="mt-6">
            <Button size="lg" onClick={() => mutation.mutate(token)} disabled={mutation.isPending}>
              {action}
            </Button>
          </div>
        </>
      )}

      <p className="mt-6 text-sm">
        <Link to={localePath('/')} className="font-semibold text-brand-ink hover:underline">
          {t('journal:newsletter.backHome')}
        </Link>
      </p>
    </div>
  )
}
