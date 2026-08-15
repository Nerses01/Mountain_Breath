import { Link } from 'react-router'
import { Trans, useTranslation } from 'react-i18next'
import { useCart, useMe, useSetCartItem, useWishlist } from '../api/hooks'
import { ProductCard } from '../components/ProductCard'
import { useLocale } from '../i18n/useLocale'
import type { Product } from '../api/types'

/**
 * /wishlist — the saved shelf. The same ProductCard as the shop grid (a
 * wishlist is a grid of cards the customer picked), with the same Add
 * behaviour, and each card's own heart is how a row leaves this page —
 * un-hearting IS removal, no second delete control needed.
 */
export function WishlistPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const me = useMe()
  const wishlist = useWishlist(!!me.data)
  const cart = useCart(!!me.data)
  const setItem = useSetCartItem()

  if (me.isPending || (me.data && wishlist.isPending)) {
    return <Shell>{t('common:state.loading')}</Shell>
  }
  if (!me.data) {
    return (
      <Shell>
        <p className="text-ink-body">
          <Trans
            i18nKey="account:wishlist.signInRequired"
            components={[
              <span key="0" />,
              <Link
                key="1"
                to={localePath('/login')}
                className="font-semibold text-brand-ink hover:underline"
              />,
            ]}
          />
        </p>
      </Shell>
    )
  }
  if (wishlist.isError) {
    return (
      <Shell>
        <p className="text-danger">{t('common:state.loadFailed')}</p>
      </Shell>
    )
  }

  const products = wishlist.data ?? []

  // The card's Add: same rule as the shop page — cheapest in-stock variant,
  // one more than whatever the cart already holds.
  function addToCart(product: Product) {
    const variant = product.variants.find((v) => v.stock_qty > 0)
    if (!variant) return
    const inCart = cart.data?.items.find((it) => it.variant_id === variant.id)?.qty ?? 0
    setItem.mutate({ variantId: variant.id, qty: inCart + 1 })
  }

  return (
    <Shell>
      <h1 className="font-display text-display-md font-extrabold text-ink">
        {t('account:wishlist.title')}
      </h1>

      {products.length === 0 ? (
        <p className="mt-4 text-ink-body">
          {t('account:wishlist.empty')}{' '}
          <Link to={localePath('/shop')} className="font-semibold text-brand-ink hover:underline">
            {t('cart:browse')}
          </Link>
        </p>
      ) : (
        <div className="mt-7 grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {products.map((p) => (
            <ProductCard key={p.id} product={p} onAdd={addToCart} />
          ))}
        </div>
      )}
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-360 px-6 py-10 lg:px-14">{children}</div>
}
