import { Link } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useQuickAdd, useWishlist } from '../api/hooks'
import { ProductCard } from '../components/ProductCard'
import { useLocale } from '../i18n/useLocale'

/**
 * /account/wishlist — the saved shelf as an account pane (A1).
 *
 * The signed-in guard moved to AccountLayout. Still the shop's ProductCard
 * with the heart as removal (un-hearting IS removal); A3 replaces the card
 * with canvas 08's own (saved date, worth total, add-all).
 */
export function WishlistPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const wishlist = useWishlist(true)
  const quickAdd = useQuickAdd()

  if (wishlist.isPending) {
    return <p className="text-ink-body">{t('common:state.loading')}</p>
  }
  if (wishlist.isError) {
    return <p className="text-danger">{t('common:state.loadFailed')}</p>
  }

  const products = wishlist.data ?? []

  return (
    <>
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
        <div className="mt-7 grid gap-5 sm:grid-cols-2 xl:grid-cols-3">
          {products.map((p) => (
            <ProductCard key={p.id} product={p} onAdd={quickAdd} />
          ))}
        </div>
      )}
    </>
  )
}
