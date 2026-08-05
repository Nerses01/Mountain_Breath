import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router'
import { useTranslation } from 'react-i18next'
import { ApiError } from '../api/client'
import { useCart, useMe, useProduct, useSetCartItem } from '../api/hooks'
import { useLocale } from '../i18n/useLocale'
import { formatPrice } from '../lib/format'

export function ProductPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  // useParams reads the :slug segment from the URL /products/:slug
  const { slug } = useParams<{ slug: string }>()
  const product = useProduct(slug ?? '')

  // Which variant the user picked; null = "none yet, default to first".
  const [variantId, setVariantId] = useState<number | null>(null)

  const me = useMe()
  const cart = useCart(!!me.data)
  const addToCart = useSetCartItem()
  const navigate = useNavigate()

  if (product.isPending) {
    return <PageShell>{t('common:state.loading')}</PageShell>
  }

  if (product.isError) {
    const notFound =
      product.error instanceof ApiError && product.error.status === 404
    return (
      <PageShell>
        <p className="rounded-lg bg-red-50 p-4 text-red-600">
          {notFound ? t('catalog:notFound') : t('common:state.loadFailed')}
        </p>
        <Link to={localePath('/')} className="mt-4 inline-block text-emerald-700 underline">
          {t('catalog:back')}
        </Link>
      </PageShell>
    )
  }

  const p = product.data
  const selected =
    p.variants.find((v) => v.id === variantId) ?? p.variants[0]

  return (
    <PageShell>
      <Link to={localePath('/')} className="text-sm text-stone-400 hover:text-stone-600">
        {t('catalog:back')}
      </Link>

      <div className="mt-4 rounded-xl border border-stone-200 bg-white p-6">
        {p.image_url && (
          <img
            src={p.image_url}
            alt={p.name}
            className="mb-4 h-64 w-full rounded-lg object-cover"
          />
        )}
        <h2 className="text-2xl font-bold text-stone-800">{p.name}</h2>
        <p className="mt-2 text-stone-500">{p.description}</p>

        {selected && (
          <>
            <div className="mt-6">
              <p className="text-sm font-medium text-stone-600">{t('catalog:size')}</p>
              <div className="mt-2 flex flex-wrap gap-2">
                {p.variants.map((v) => (
                  <button
                    key={v.id}
                    type="button"
                    onClick={() => setVariantId(v.id)}
                    disabled={v.stock_qty === 0}
                    className={
                      v.id === selected.id
                        ? 'rounded-lg bg-emerald-700 px-4 py-2 text-sm font-medium text-white'
                        : 'rounded-lg bg-white px-4 py-2 text-sm font-medium text-stone-600 ring-1 ring-stone-200 hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-40'
                    }
                  >
                    {v.label}
                  </button>
                ))}
              </div>
            </div>

            <div className="mt-6 flex items-end justify-between">
              <div>
                <p className="text-3xl font-bold text-stone-800">
                  {formatPrice(selected.price_minor)}
                </p>
                <p className="mt-1 text-xs text-stone-400">
                  {selected.stock_qty > 0
                    ? `${t('catalog:inStock', { count: selected.stock_qty })} · ${selected.sku}`
                    : t('catalog:outOfStock')}
                </p>
              </div>
              <AddToCartButton
                inCartQty={
                  cart.data?.items.find((it) => it.variant_id === selected.id)?.qty ?? 0
                }
                outOfStock={selected.stock_qty === 0}
                loggedIn={!!me.data}
                isPending={addToCart.isPending}
                onAdd={() =>
                  me.data
                    ? addToCart.mutate({ variantId: selected.id, qty: 1 })
                    : navigate(localePath('/login'))
                }
              />
            </div>
          </>
        )}
      </div>
    </PageShell>
  )
}

function AddToCartButton({
  inCartQty,
  outOfStock,
  loggedIn,
  isPending,
  onAdd,
}: {
  inCartQty: number
  outOfStock: boolean
  loggedIn: boolean
  isPending: boolean
  onAdd: () => void
}) {
  const { t } = useTranslation()
  const { localePath } = useLocale()

  if (outOfStock) {
    return (
      <button
        type="button"
        disabled
        className="cursor-not-allowed rounded-lg bg-stone-300 px-6 py-3 font-medium text-white"
      >
        {t('catalog:outOfStock')}
      </button>
    )
  }

  if (inCartQty > 0) {
    return (
      <Link
        to={localePath('/cart')}
        className="rounded-lg bg-stone-800 px-6 py-3 font-medium text-white hover:bg-stone-700"
      >
        {t('catalog:inCart', { count: inCartQty })} →
      </Link>
    )
  }

  return (
    <button
      type="button"
      disabled={isPending}
      onClick={onAdd}
      className="rounded-lg bg-emerald-700 px-6 py-3 font-medium text-white hover:bg-emerald-800 disabled:opacity-50"
    >
      {loggedIn
        ? isPending
          ? t('catalog:adding')
          : t('catalog:addToCart')
        : t('catalog:signInToBuy')}
    </button>
  )
}

function PageShell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-3xl px-4 py-8">{children}</div>
}
