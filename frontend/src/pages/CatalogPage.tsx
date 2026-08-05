import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useCategories, useProducts } from '../api/hooks'
import { ProductCard } from '../components/ProductCard'
import { useDebouncedValue } from '../lib/useDebounce'

export function CatalogPage() {
  const { t } = useTranslation()
  // '' means "all categories"; changing this state re-renders the page and
  // useProducts fetches (or serves from cache) the matching list.
  const [category, setCategory] = useState('')
  const [search, setSearch] = useState('')
  // The input updates instantly (responsive typing); the QUERY only fires
  // when typing pauses — debounced value is part of the query key.
  const debouncedSearch = useDebouncedValue(search, 300)

  const categories = useCategories()
  const products = useProducts({ category, q: debouncedSearch })

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <input
        type="search"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder={t('catalog:searchPlaceholder')}
        aria-label={t('common:actions.search')}
        className="mb-4 w-full rounded-xl border border-stone-300 bg-white px-4 py-2.5 text-sm focus:border-emerald-600 focus:outline-none"
      />

      <nav className="flex flex-wrap gap-2">
        <FilterChip
          label={t('catalog:all')}
          active={category === ''}
          onClick={() => setCategory('')}
        />
        {categories.data?.map((c) => (
          <FilterChip
            key={c.id}
            label={c.name}
            active={category === c.slug}
            onClick={() => setCategory(c.slug)}
          />
        ))}
      </nav>

      <main className="mt-6">
        {products.isPending && (
          <p className="text-stone-400">{t('common:state.loading')}</p>
        )}

        {products.isError && (
          <p className="rounded-lg bg-red-50 p-4 text-red-600">
            {t('common:state.loadFailed')}
          </p>
        )}

        {products.data && (
          <>
            <p className="text-sm text-stone-400">
              {/* The count drives plural selection, so Russian gets
                  продукт/продукта/продуктов rather than an English rule
                  applied to a Russian word. */}
              {debouncedSearch
                ? t('catalog:resultsFor', {
                    count: products.data.total,
                    query: debouncedSearch,
                  })
                : t('catalog:resultCount', { count: products.data.total })}
            </p>
            <div className="mt-3 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3">
              {products.data.items.map((p) => (
                <ProductCard key={p.id} product={p} />
              ))}
            </div>
          </>
        )}
      </main>
    </div>
  )
}

function FilterChip({
  label,
  active,
  onClick,
}: {
  label: string
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={
        active
          ? 'rounded-full bg-emerald-700 px-4 py-1.5 text-sm font-medium text-white'
          : 'rounded-full bg-white px-4 py-1.5 text-sm font-medium text-stone-600 ring-1 ring-stone-200 hover:bg-stone-50'
      }
    >
      {label}
    </button>
  )
}
