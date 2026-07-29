import { Link } from 'react-router'
import { useCart, useMe } from '../api/hooks'

export function CartLink() {
  const me = useMe()
  const cart = useCart(!!me.data)

  if (!me.data) return null // cart is a logged-in feature

  const count = cart.data?.items.reduce((sum, it) => sum + it.qty, 0) ?? 0

  return (
    <Link to="/cart" className="relative text-sm font-medium text-stone-600 hover:text-stone-800">
      🧺 Cart
      {count > 0 && (
        <span className="absolute -top-2 -right-3 rounded-full bg-emerald-700 px-1.5 text-xs font-bold text-white">
          {count}
        </span>
      )}
    </Link>
  )
}
