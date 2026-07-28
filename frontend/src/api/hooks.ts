import { useQuery } from '@tanstack/react-query'
import { api, type ProductListParams } from './client'

// TanStack Query caches by queryKey: two components asking for the same key
// share one request and one cached result.

export function useCategories() {
  return useQuery({
    queryKey: ['categories'],
    queryFn: api.listCategories,
  })
}

export function useProducts(params: ProductListParams) {
  return useQuery({
    // params are part of the key — changing the category filter is a new
    // cache entry, and going back to a seen filter is instant.
    queryKey: ['products', params],
    queryFn: () => api.listProducts(params),
  })
}
