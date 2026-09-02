import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ApiError } from '../../api/client'
import { useCreateReview, useReviews } from '../../api/hooks'
import type { ProductDetail } from '../../api/types'
import { useFieldErrors } from '../../i18n/useFieldErrors'
import { cx } from '../../lib/cx'
import { Button, Card, Stars } from '../ui'

const PER_PAGE = 10

/**
 * The Reviews tab: the published list, and — for someone who has actually
 * received the product — the form.
 *
 * The design draws neither (it shows only the tab label and a count), so the
 * whole panel is §6 exception 2 territory.
 */
export function Reviews({ product }: { product: ProductDetail }) {
  const { t } = useTranslation()
  const [page, setPage] = useState(1)
  const reviews = useReviews(product.slug, page)

  const pageCount = Math.max(1, Math.ceil((reviews.data?.total ?? 0) / PER_PAGE))

  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-wrap items-center gap-4">
        <Stars rating={product.rating_avg} count={product.rating_count} size="md" />
      </div>

      {product.can_review && <ReviewForm product={product} />}

      {reviews.isPending && <p className="text-ink-muted">{t('common:state.loading')}</p>}

      {reviews.data && reviews.data.total === 0 && (
        <p className="text-base text-ink-soft">{t('product:reviews.empty')}</p>
      )}

      <ul className="flex flex-col gap-4">
        {reviews.data?.items.map((r) => (
          <li key={r.id}>
            <Card className="flex flex-col gap-2 p-5">
              <div className="flex flex-wrap items-center gap-3">
                <Stars rating={r.rating} showCount={false} />
                <span className="font-display text-sm font-semibold text-ink">
                  {r.author}
                </span>
                {/* The reader's locale formats the date, so 2026-08-11 is
                    11.08.2026 in Russian and 08/11/2026 in English without
                    this component knowing anything about either. */}
                <time
                  dateTime={r.created_at}
                  className="text-xs text-ink-muted"
                >
                  {new Date(r.created_at).toLocaleDateString(undefined, {
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric',
                  })}
                </time>
              </div>
              {r.title && (
                <h3 className="font-display text-base font-bold text-ink">{r.title}</h3>
              )}
              {r.body && <p className="text-base leading-relaxed text-ink-body">{r.body}</p>}
            </Card>
          </li>
        ))}
      </ul>

      {pageCount > 1 && (
        <nav aria-label={t('product:reviews.pagination')} className="flex gap-2.5">
          {Array.from({ length: pageCount }, (_, i) => i + 1).map((n) => (
            <button
              key={n}
              type="button"
              aria-current={n === page ? 'page' : undefined}
              onClick={() => setPage(n)}
              className={cx(
                'inline-flex size-9 items-center justify-center rounded-full font-display text-sm',
                n === page
                  ? 'bg-brand-ink font-bold text-ink-on-dark'
                  : 'border-[1.5px] border-line text-ink-body hover:border-line-strong',
              )}
            >
              {n}
            </button>
          ))}
        </nav>
      )}
    </div>
  )
}

function ReviewForm({ product }: { product: ProductDetail }) {
  const { t } = useTranslation()
  const create = useCreateReview(product.slug)
  const { fieldError, hasFormError } = useFieldErrors(create.error)

  const [rating, setRating] = useState(0)
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')

  if (create.isSuccess) {
    // Nothing appears in the list, because the review is pending. Saying so
    // is the difference between "we received it" and a reader concluding the
    // form is broken.
    return (
      <Card tone="panel" className="p-5">
        <p className="text-base text-ink-body">{t('product:reviews.thanks')}</p>
      </Card>
    )
  }

  return (
    <Card className="flex flex-col gap-4 p-5">
      <h3 className="font-display text-base font-bold text-ink">
        {t('product:reviews.formTitle')}
      </h3>

      <form
        className="flex flex-col gap-4"
        onSubmit={(e) => {
          e.preventDefault()
          create.mutate({ rating, title, body })
        }}
      >
        {/* A radiogroup, not five buttons: choosing a rating is choosing ONE
            of a set, and the browser gives arrow-key navigation between
            radios for free. The visible control is the star; the radio
            itself is the accessible one, hidden with sr-only rather than
            display:none, which would take it out of the tab order. */}
        <fieldset>
          <legend className="mb-2 font-display text-sm font-semibold text-ink">
            {t('product:reviews.yourRating')}
          </legend>
          <div className="flex gap-1">
            {[1, 2, 3, 4, 5].map((n) => (
              <label
                key={n}
                className={cx(
                  'cursor-pointer rounded p-1 transition',
                  'focus-within:outline focus-within:outline-2 focus-within:outline-brand-ink',
                )}
              >
                <input
                  type="radio"
                  name="rating"
                  value={n}
                  checked={rating === n}
                  onChange={() => setRating(n)}
                  className="sr-only"
                />
                <span className="sr-only">
                  {t('product:rating.outOf', { rating: n })}
                </span>
                <Stars rating={rating >= n ? 5 : 0} showCount={false} size="md" />
              </label>
            ))}
          </div>
          {fieldError('rating') && (
            <p className="mt-1 text-sm text-danger">{fieldError('rating')}</p>
          )}
        </fieldset>

        <label className="flex flex-col gap-1">
          <span className="font-display text-sm font-semibold text-ink">
            {t('product:reviews.title')}
          </span>
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            maxLength={120}
            className="rounded-md border-[1.5px] border-line bg-card px-3 py-2 text-base"
          />
          {fieldError('title') && (
            <span className="text-sm text-danger">{fieldError('title')}</span>
          )}
        </label>

        <label className="flex flex-col gap-1">
          <span className="font-display text-sm font-semibold text-ink">
            {t('product:reviews.body')}
          </span>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={4}
            maxLength={4000}
            className="rounded-md border-[1.5px] border-line bg-card px-3 py-2 text-base"
          />
          {fieldError('body') && (
            <span className="text-sm text-danger">{fieldError('body')}</span>
          )}
        </label>

        {/* A 403 here means the verified-purchase rule refused — which
            can_review should have prevented, so it is worth saying out loud
            rather than swallowing. */}
        {create.isError && hasFormError && (
          <p className="text-sm text-danger">
            {create.error instanceof ApiError && create.error.status === 403
              ? t('product:reviews.notPurchased')
              : t('common:state.loadFailed')}
          </p>
        )}

        <Button type="submit" disabled={rating === 0 || create.isPending} className="self-start">
          {create.isPending ? t('catalog:adding') : t('product:reviews.submit')}
        </Button>
      </form>
    </Card>
  )
}
