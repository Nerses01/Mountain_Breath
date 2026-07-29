import { Link } from 'react-router'
import { useAdminOrders, useMe, useUpdateOrderStatus } from '../api/hooks'
import { OrderCard } from '../components/OrderCard'
import type { OrderStatus } from '../api/types'

// Mirror of the backend's state machine — for showing the right buttons.
// The backend remains the enforcer; this is UX.
const nextStatuses: Record<OrderStatus, OrderStatus[]> = {
  pending: ['confirmed', 'cancelled'],
  confirmed: ['shipped', 'cancelled'],
  shipped: ['delivered'],
  delivered: [],
  cancelled: [],
}

export function AdminOrdersPage() {
  const me = useMe()
  const orders = useAdminOrders()
  const update = useUpdateOrderStatus()

  if (me.isPending) return <Shell>Checking access…</Shell>
  if (!me.data || me.data.role !== 'admin') {
    return (
      <Shell>
        <p className="rounded-lg bg-red-50 p-4 text-red-600">
          This area requires an admin account.
        </p>
      </Shell>
    )
  }

  return (
    <Shell>
      <div className="flex items-center gap-4">
        <h2 className="text-xl font-bold text-stone-800">Admin — Orders</h2>
        <Link to="/admin" className="text-sm text-emerald-700 hover:underline">
          → Categories
        </Link>
      </div>

      {orders.isPending && <p className="mt-4 text-stone-400">Loading…</p>}
      {orders.isError && <p className="mt-4 text-red-600">Failed to load orders.</p>}

      {orders.data && (
        <div className="mt-4 space-y-4">
          {orders.data.length === 0 && <p className="text-stone-500">No orders yet.</p>}
          {orders.data.map((o) => (
            <OrderCard key={o.id} order={o}>
              {nextStatuses[o.status].length > 0 && (
                <div className="mt-3 flex gap-2 border-t border-stone-100 pt-3">
                  {nextStatuses[o.status].map((next) => (
                    <button
                      key={next}
                      type="button"
                      disabled={update.isPending}
                      onClick={() => update.mutate({ orderId: o.id, status: next })}
                      className={
                        next === 'cancelled'
                          ? 'rounded-lg bg-red-50 px-4 py-1.5 text-sm font-medium text-red-600 hover:bg-red-100 disabled:opacity-50'
                          : 'rounded-lg bg-emerald-700 px-4 py-1.5 text-sm font-medium text-white hover:bg-emerald-800 disabled:opacity-50'
                      }
                    >
                      mark {next}
                    </button>
                  ))}
                </div>
              )}
            </OrderCard>
          ))}
        </div>
      )}
    </Shell>
  )
}

function Shell({ children }: { children: React.ReactNode }) {
  return <div className="mx-auto max-w-3xl px-4 py-8">{children}</div>
}
