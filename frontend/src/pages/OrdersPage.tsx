import { useTranslation } from 'react-i18next'
import { useMyOrders } from '../api/hooks'
import { OrderCard } from '../components/OrderCard'

/**
 * /account/orders — order history as an account pane (A1).
 *
 * The signed-in guard and page shell moved to AccountLayout; what remains is
 * the list itself. A2 rebuilds this into canvas 07 (filter pills, the active
 * order's tracker, compact history rows).
 */
export function OrdersPage() {
  const { t } = useTranslation()
  const orders = useMyOrders()

  if (orders.isPending) {
    return <p className="text-ink-body">{t('common:state.loading')}</p>
  }
  if (orders.isError) {
    return <p className="text-danger">{t('common:state.loadFailed')}</p>
  }

  return (
    <>
      <h1 className="font-display text-display-md font-extrabold text-ink">
        {t('account:ordersTitle')}
      </h1>
      {orders.data.length === 0 ? (
        <p className="mt-4 text-ink-body">{t('account:noOrders')}</p>
      ) : (
        <div className="mt-7 space-y-4">
          {orders.data.map((o) => (
            <OrderCard key={o.id} order={o} />
          ))}
        </div>
      )}
    </>
  )
}
