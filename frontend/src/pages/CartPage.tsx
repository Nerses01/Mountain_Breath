import { Link, useNavigate } from 'react-router'
import { Trans, useTranslation } from 'react-i18next'
import { useCart, useCheckout, useMe, useRemoveCartItem, useSetCartItem } from '../api/hooks'
import { useFieldErrors } from '../i18n/useFieldErrors'
import { useLocale } from '../i18n/useLocale'
import { formatPrice } from '../lib/format'

export function CartPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const me = useMe()
  const cart = useCart(!!me.data)
  const setItem = useSetCartItem()
  const removeItem = useRemoveCartItem()
  const checkout = useCheckout()
  const navigate = useNavigate()
  const { formError } = useFieldErrors(checkout.error)

  if (me.isPending || cart.isPending) {
    return <Shell>{t('common:state.loading')}</Shell>
  }
  if (!me.data) {
    return (
      <Shell>
        <p className="text-stone-500">
          {/* <Trans> renders a sentence that CONTAINS a link. Splitting it
              into "Please " + link + " to use the cart" would be untranslatable:
              word order differs per language, and Armenian puts the verb last.
              The <1> placeholder in the message says where the link goes. */}
          <Trans
            i18nKey="cart:signInRequired"
            components={[
              <span key="0" />,
              <Link key="1" to={localePath('/login')} className="text-emerald-700 underline" />,
            ]}
          />
        </p>
      </Shell>
    )
  }
  if (cart.isError) {
    return <Shell><p className="text-red-600">{t('common:state.loadFailed')}</p></Shell>
  }

  const items = cart.data?.items ?? []

  return (
    <Shell>
      <h2 className="text-xl font-bold text-stone-800">{t('cart:title')}</h2>

      {items.length === 0 && (
        <p className="mt-4 text-stone-500">
          {t('cart:empty')}{' '}
          <Link to={localePath('/')} className="text-emerald-700 underline">
            {t('cart:browse')}
          </Link>
        </p>
      )}

      {items.length > 0 && (
        <>
          <ul className="mt-4 space-y-3">
            {items.map((it) => (
              <li
                key={it.variant_id}
                className="flex flex-wrap items-center gap-3 rounded-xl border border-stone-200 bg-white p-4"
              >
                <div className="min-w-40 flex-1">
                  <Link
                    to={`/products/${it.product_slug}`}
                    className="font-medium text-stone-800 hover:text-emerald-800"
                  >
                    {it.product_name}
                  </Link>
                  <p className="text-xs text-stone-400">
                    {it.label} · {formatPrice(it.price_minor)} {t('cart:each')}
                  </p>
                </div>

                <div className="flex items-center gap-2">
                  <QtyButton
                    label="−"
                    title={t('cart:decrease')}
                    onClick={() =>
                      it.qty <= 1
                        ? removeItem.mutate(it.variant_id)
                        : setItem.mutate({ variantId: it.variant_id, qty: it.qty - 1 })
                    }
                  />
                  <span className="w-8 text-center font-medium">{it.qty}</span>
                  <QtyButton
                    label="+"
                    title={t('cart:increase')}
                    disabled={it.qty >= it.stock_qty}
                    onClick={() => setItem.mutate({ variantId: it.variant_id, qty: it.qty + 1 })}
                  />
                </div>

                <span className="w-24 text-right font-semibold text-stone-800">
                  {formatPrice(it.line_total_minor)}
                </span>

                <button
                  type="button"
                  onClick={() => removeItem.mutate(it.variant_id)}
                  className="text-stone-300 hover:text-red-500"
                  title={t('cart:remove')}
                  aria-label={t('cart:remove')}
                >
                  ✕
                </button>
              </li>
            ))}
          </ul>

          {formError && (
            <p className="mt-4 rounded-lg bg-red-50 p-3 text-sm text-red-600">
              {formError}
            </p>
          )}

          <div className="mt-6 flex items-center justify-between rounded-xl border border-stone-200 bg-white p-4">
            <div>
              <p className="text-sm text-stone-500">{t('cart:total')}</p>
              <p className="text-2xl font-bold text-stone-800">
                {formatPrice(cart.data?.total_minor ?? 0)}
              </p>
            </div>
            <button
              type="button"
              disabled={checkout.isPending}
              onClick={() =>
                checkout.mutate(undefined, { onSuccess: () => navigate('/orders') })
              }
              className="rounded-lg bg-emerald-700 px-8 py-3 font-medium text-white hover:bg-emerald-800 disabled:opacity-50"
            >
              {checkout.isPending ? t('cart:placingOrder') : t('cart:checkout')}
            </button>
          </div>
        </>
      )}
    </Shell>
  )
}

function QtyButton({
  label,
  title,
  onClick,
  disabled,
}: {
  label: string
  title: string
  onClick: () => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      // The glyph is decoration; the accessible name comes from the title.
      aria-label={title}
      title={title}
      className="h-8 w-8 rounded-lg bg-stone-100 font-bold text-stone-600 hover:bg-stone-200 disabled:opacity-40"
    >
      {label}
    </button>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-3xl px-4 py-8">{children}</div>
}
