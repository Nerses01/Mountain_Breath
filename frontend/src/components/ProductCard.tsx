import { Link } from 'react-router'
import type { Product } from '../api/types'
import { formatPrice } from '../lib/format'

export function ProductCard({ product }: { product: Product }) {
  return (
    <Link to={`/products/${product.slug}`} className="group">
      <article className="flex h-full flex-col rounded-xl border border-stone-200 bg-white p-5 shadow-sm transition-shadow group-hover:shadow-md">
        <h3 className="text-lg font-semibold text-stone-800 group-hover:text-emerald-800">
          {product.name}
        </h3>
        <p className="mt-1 flex-1 text-sm text-stone-500">{product.description}</p>

        <ul className="mt-4 space-y-2">
          {product.variants.map((v) => (
            <li
              key={v.id}
              className="flex items-center justify-between rounded-lg bg-stone-50 px-3 py-2 text-sm"
            >
              <span className="font-medium text-stone-700">{v.label}</span>
              <span className="flex items-center gap-3">
                {v.stock_qty === 0 ? (
                  <span className="text-xs text-red-500">out of stock</span>
                ) : (
                  <span className="text-xs text-stone-400">{v.stock_qty} left</span>
                )}
                <span className="font-semibold text-stone-800">
                  {formatPrice(v.price_minor)}
                </span>
              </span>
            </li>
          ))}
        </ul>
      </article>
    </Link>
  )
}
