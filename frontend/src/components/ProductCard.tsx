import type { Product } from '../api/types'

// Money arrives as integer minor units (e.g. 350000 = 3500.00) — format at
// the last moment, only for display.
function formatPrice(priceMinor: number): string {
  return (priceMinor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  })
}

export function ProductCard({ product }: { product: Product }) {
  return (
    <article className="flex flex-col rounded-xl border border-stone-200 bg-white p-5 shadow-sm transition-shadow hover:shadow-md">
      <h3 className="text-lg font-semibold text-stone-800">{product.name}</h3>
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
  )
}
