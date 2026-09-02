import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Preview } from '../../api/types'
import { useApplyPromo, useRemovePromo } from '../../api/hooks'
import { useFieldErrors } from '../../i18n/useFieldErrors'
import { cx } from '../../lib/cx'

/**
 * The design's "Promo code" card. Two states the mock draws one of:
 *
 *  - empty: input + Apply. Every way a code can be wrong comes back as a
 *    `fields.promo_code` validation CODE and renders as a sentence under
 *    the input — the same envelope, catalogue and hook as any other form.
 *  - applied: the code as a pill with a remove button. If the code has
 *    STOPPED applying since (expired, basket shrank), the preview says why
 *    in `promo_issue` and the box complains about the code by name instead
 *    of silently dropping a discount the customer believes they have.
 */
export function PromoBox({ preview }: { preview: Preview }) {
  const { t } = useTranslation()
  const [code, setCode] = useState('')
  const apply = useApplyPromo()
  const remove = useRemovePromo()
  const errors = useFieldErrors(apply.error)

  const applied = preview.promo_code

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!code.trim()) return
    apply.mutate(code, { onSuccess: () => setCode('') })
  }

  return (
    <section className="flex flex-col gap-2.5 rounded-2xl bg-card p-6">
      <h2 className="font-display text-[0.9375rem] font-bold uppercase tracking-label text-ink">
        {t('cart:promo.title')}
      </h2>

      {applied ? (
        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between gap-2">
            <span className="rounded-full bg-panel px-4 py-2 font-display text-sm font-bold tracking-wide text-ink">
              {t('cart:promo.applied', { code: applied })}
            </span>
            <button
              type="button"
              onClick={() => remove.mutate()}
              disabled={remove.isPending}
              className="text-sm font-semibold text-brand-ink hover:underline disabled:opacity-50"
            >
              {t('cart:promo.remove')}
            </button>
          </div>
          {preview.promo_issue && (
            <p role="alert" className="text-xs text-danger">
              {t([`validation:${preview.promo_issue}`, 'validation:unknown'])}
            </p>
          )}
        </div>
      ) : (
        <form onSubmit={onSubmit} className="flex flex-col gap-2" noValidate>
          <div className="flex gap-2">
            <input
              value={code}
              onChange={(e) => {
                setCode(e.target.value)
                apply.reset() // editing clears the last attempt's error
              }}
              placeholder={t('cart:promo.placeholder')}
              aria-label={t('cart:promo.title')}
              aria-invalid={Boolean(errors.fieldError('promo_code')) || undefined}
              className={cx(
                'min-w-0 flex-1 rounded-full border-[1.5px] bg-card px-4 py-3 text-sm text-ink',
                'placeholder:text-ink-faint',
                errors.fieldError('promo_code') ? 'border-danger' : 'border-line',
              )}
            />
            <button
              type="submit"
              disabled={apply.isPending || !code.trim()}
              className="rounded-full bg-bark px-5 py-3 font-display text-sm font-semibold text-ink-on-dark transition hover:bg-bark-soft disabled:opacity-50"
            >
              {apply.isPending ? t('cart:promo.applying') : t('cart:promo.apply')}
            </button>
          </div>
          {errors.fieldError('promo_code') && (
            <p role="alert" className="text-xs text-danger">
              {errors.fieldError('promo_code')}
            </p>
          )}
        </form>
      )}
    </section>
  )
}
