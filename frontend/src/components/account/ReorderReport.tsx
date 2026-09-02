import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import type { ReorderResult } from '../../api/types'
import { useLocale } from '../../i18n/useLocale'

/**
 * What a cart merge did — reorder (A2) and the wishlist's add-all (A3)
 * share the endpoint contract, so they share the banner: a polite
 * announcement of what was added, with every skip named and its reason
 * translated from the server's issue code.
 */
export function ReorderReport({ report }: { report: ReorderResult }) {
  const { t } = useTranslation()
  const { localePath } = useLocale()

  const added = report.lines.reduce((sum, l) => sum + l.qty, 0)
  const issues = report.lines.filter((l) => l.issue)

  return (
    <div role="status" className="mt-5 flex flex-col gap-1.5 rounded-2xl bg-honey/25 px-5 py-4">
      <p className="text-sm font-semibold text-ink">
        {added > 0
          ? t('account:ordersScreen.reorderAdded', { count: added })
          : t('account:ordersScreen.reorderNothing')}{' '}
        {added > 0 && (
          <Link to={localePath('/cart')} className="font-semibold text-brand-ink hover:underline">
            {t('account:ordersScreen.viewCart')}
          </Link>
        )}
      </p>
      {issues.map((l) => (
        <p key={l.name + l.label} className="text-[0.8125rem] text-ink-body">
          {l.name}
          {l.label && ` (${l.label})`} — {t(`account:ordersScreen.issue.${l.issue}`)}
        </p>
      ))}
    </div>
  )
}
