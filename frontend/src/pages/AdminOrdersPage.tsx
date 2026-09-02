import { useAdminOrders, useMe, useUpdateOrderPayment, useUpdateOrderStatus } from '../api/hooks'
import { AdminNav } from '../components/AdminNav'
import { OrderCard } from '../components/OrderCard'
import type { OrderStatus, PaymentStatus } from '../api/types'

// Mirror of the backend's state machine — for showing the right buttons.
// The backend remains the enforcer; this is UX.
const nextStatuses: Record<OrderStatus, OrderStatus[]> = {
  pending: ['confirmed', 'cancelled'],
  confirmed: ['shipped', 'cancelled'],
  shipped: ['delivered'],
  delivered: [],
  cancelled: [],
}

// F2: the payment machine's mirror, same contract. No backward arrow —
// a mistaken "paid" is corrected by the compensating "refunded", never
// erased, so the table reads exactly like the domain's.
const nextPayments: Record<PaymentStatus, PaymentStatus[]> = {
  unpaid: ['paid'],
  paid: ['refunded'],
  refunded: [],
}

const paymentStyles: Record<PaymentStatus, string> = {
  unpaid: 'bg-amber-100 text-amber-800',
  paid: 'bg-emerald-100 text-emerald-800',
  refunded: 'bg-stone-200 text-stone-500',
}

export function AdminOrdersPage() {
  const me = useMe()
  const orders = useAdminOrders()
  const update = useUpdateOrderStatus()
  const updatePayment = useUpdateOrderPayment()

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
      <div className="flex items-center gap-6">
        <h2 className="text-xl font-bold text-stone-800">Admin — Orders</h2>
        <AdminNav />
      </div>

      {orders.isPending && <p className="mt-4 text-stone-400">Loading…</p>}
      {orders.isError && <p className="mt-4 text-red-600">Failed to load orders.</p>}

      {orders.data && (
        <div className="mt-4 space-y-4">
          {orders.data.length === 0 && <p className="text-stone-500">No orders yet.</p>}
          {orders.data.map((o) => (
            <OrderCard key={o.id} order={o}>
              {/* F2: the payment row — the two orthogonal machines get two
                  rows, so "where is the parcel" and "has the money arrived"
                  never read as one muddled state. */}
              <div className="mt-3 flex flex-wrap items-center gap-2 border-t border-stone-100 pt-3">
                <span className="text-sm text-stone-500">
                  {o.payment_method.replace(/_/g, ' ')}
                </span>
                <span
                  className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${paymentStyles[o.payment_status]}`}
                >
                  {o.payment_status}
                </span>
                {nextPayments[o.payment_status].map((next) => (
                  <button
                    key={next}
                    type="button"
                    disabled={updatePayment.isPending}
                    onClick={() => updatePayment.mutate({ orderId: o.id, paymentStatus: next })}
                    className={
                      next === 'refunded'
                        ? 'rounded-lg bg-amber-50 px-4 py-1.5 text-sm font-medium text-amber-700 hover:bg-amber-100 disabled:opacity-50'
                        : 'rounded-lg bg-emerald-700 px-4 py-1.5 text-sm font-medium text-white hover:bg-emerald-800 disabled:opacity-50'
                    }
                  >
                    mark {next}
                  </button>
                ))}
              </div>

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
