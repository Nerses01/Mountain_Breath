import type { Order, OrderStatus } from '../api/types'
import { formatPrice } from '../lib/format'

const statusStyles: Record<OrderStatus, string> = {
  pending: 'bg-amber-100 text-amber-800',
  confirmed: 'bg-blue-100 text-blue-800',
  shipped: 'bg-violet-100 text-violet-800',
  delivered: 'bg-emerald-100 text-emerald-800',
  cancelled: 'bg-stone-200 text-stone-500',
}

export function StatusBadge({ status }: { status: OrderStatus }) {
  return (
    <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${statusStyles[status]}`}>
      {status}
    </span>
  )
}

export function OrderCard({ order, children }: { order: Order; children?: React.ReactNode }) {
  return (
    <article className="rounded-xl border border-stone-200 bg-white p-5">
      <div className="flex flex-wrap items-center gap-3">
        <span className="font-semibold text-stone-800">Order #{order.id}</span>
        <StatusBadge status={order.status} />
        {order.user_email && <span className="text-xs text-stone-400">{order.user_email}</span>}
        <span className="ml-auto text-xs text-stone-400">
          {new Date(order.created_at).toLocaleString()}
        </span>
      </div>

      <ul className="mt-3 space-y-1 text-sm">
        {order.items.map((it, i) => (
          <li key={i} className="flex justify-between">
            <span className="text-stone-600">
              {it.qty} × {it.name} <span className="text-stone-400">({it.label})</span>
            </span>
            <span className="text-stone-700">{formatPrice(it.price_minor * it.qty)}</span>
          </li>
        ))}
      </ul>

      <div className="mt-3 flex items-center justify-between border-t border-stone-100 pt-3">
        <span className="text-sm text-stone-500">Total</span>
        <span className="font-bold text-stone-800">{formatPrice(order.total_minor)}</span>
      </div>

      {children}
    </article>
  )
}
