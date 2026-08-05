import { Link } from 'react-router'
import { Trans, useTranslation } from 'react-i18next'
import { useMe, useMyOrders } from '../api/hooks'
import { useLocale } from '../i18n/useLocale'
import { OrderCard } from '../components/OrderCard'

export function OrdersPage() {
  const { t } = useTranslation()
  const { localePath } = useLocale()
  const me = useMe()
  const orders = useMyOrders()

  if (me.isPending || orders.isPending) {
    return <Shell>{t('common:state.loading')}</Shell>
  }
  if (!me.data) {
    return (
      <Shell>
        <p className="text-stone-500">
          <Trans
            i18nKey="account:signInRequired"
            components={[
              <span key="0" />,
              <Link key="1" to={localePath('/login')} className="text-emerald-700 underline" />,
            ]}
          />
        </p>
      </Shell>
    )
  }
  if (orders.isError) {
    return <Shell><p className="text-red-600">{t('common:state.loadFailed')}</p></Shell>
  }

  return (
    <Shell>
      <h2 className="text-xl font-bold text-stone-800">{t('account:ordersTitle')}</h2>
      {orders.data.length === 0 ? (
        <p className="mt-4 text-stone-500">{t('account:noOrders')}</p>
      ) : (
        <div className="mt-4 space-y-4">
          {orders.data.map((o) => (
            <OrderCard key={o.id} order={o} />
          ))}
        </div>
      )}
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-3xl px-4 py-8">{children}</div>
}
