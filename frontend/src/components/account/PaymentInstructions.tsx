import { useTranslation } from 'react-i18next'
import type { PaymentInstructions as Instructions } from '../../api/types'
import { formatMoney } from '../../lib/format'

/**
 * "How to pay" (decision #110), drawn from what the server composed: the
 * amount and, for a transfer, the account and the purpose line. The
 * confirmation mail says the same things from the same domain pieces, so
 * this panel never invents a fact. Nothing to draw once the order is paid:
 * the server then sends no instructions at all.
 *
 * The canvas draws no such panel — its confirmation screen stops at "Card,
 * bank transfer or cash on delivery" — so the design is ours (rule #16,
 * standing exception 2): inside the payment card, a definition list for
 * the account so a screen reader pairs each label with its value, and the
 * account number and purpose in a monospace face because they get copied
 * by hand. "Account", not "IBAN": Armenia is not in the IBAN registry, and
 * the field holds whatever the bank prints.
 */
export function PaymentInstructions({ instructions }: { instructions?: Instructions }) {
  const { t } = useTranslation()
  if (!instructions) return null
  const amount = formatMoney(instructions.amount_minor, instructions.currency)

  if (instructions.method === 'cash_on_delivery') {
    return <p className="mt-3 text-sm text-ink-body">{t('order:howToPay.cash', { amount })}</p>
  }
  if (instructions.method !== 'bank_transfer') return null

  if (!instructions.bank) {
    return <p className="mt-3 text-sm text-ink-body">{t('order:howToPay.bankLater', { amount })}</p>
  }
  return (
    <div className="mt-3 flex flex-col gap-2 text-sm text-ink-body">
      <p>{t('order:howToPay.bankIntro', { amount })}</p>
      <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1">
        <dt className="text-xs text-ink-soft">{t('order:howToPay.recipient')}</dt>
        <dd>{instructions.bank.recipient}</dd>
        <dt className="text-xs text-ink-soft">{t('order:howToPay.bank')}</dt>
        <dd>{instructions.bank.bank}</dd>
        <dt className="text-xs text-ink-soft">{t('order:howToPay.account')}</dt>
        <dd className="font-mono">{instructions.bank.account}</dd>
        <dt className="text-xs text-ink-soft">{t('order:howToPay.reference')}</dt>
        <dd className="font-mono">{instructions.reference}</dd>
      </dl>
      <p className="text-xs text-ink-soft">{t('order:howToPay.bankShips')}</p>
    </div>
  )
}
