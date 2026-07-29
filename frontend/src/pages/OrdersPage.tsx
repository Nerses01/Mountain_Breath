import { Link } from 'react-router'
import { useMe, useMyOrders } from '../api/hooks'
import { OrderCard } from '../components/OrderCard'

export function OrdersPage() {
  const me = useMe()
  const orders = useMyOrders()

  if (me.isPending || orders.isPending) {
    return <Shell>Loading…</Shell>
  }
  if (!me.data) {
    return (
      <Shell>
        <p className="text-stone-500">
          Please{' '}
          <Link to="/login" className="text-emerald-700 underline">
            sign in
          </Link>{' '}
          to see your orders.
        </p>
      </Shell>
    )
  }
  if (orders.isError) {
    return <Shell><p className="text-red-600">Failed to load orders.</p></Shell>
  }

  return (
    <Shell>
      <h2 className="text-xl font-bold text-stone-800">Your orders</h2>
      {orders.data.length === 0 ? (
        <p className="mt-4 text-stone-500">No orders yet.</p>
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
