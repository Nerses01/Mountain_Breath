import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, ApiError, type ProductListParams } from './client'
import type { User } from './types'

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

export function useProduct(slug: string) {
  return useQuery({
    queryKey: ['product', slug],
    queryFn: () => api.getProduct(slug),
  })
}

// useMe maps 401 to `null` ("nobody is logged in") instead of an error —
// being anonymous is a normal state, not a failure.
export function useMe() {
  return useQuery<User | null>({
    queryKey: ['me'],
    queryFn: async () => {
      try {
        return await api.me()
      } catch (e) {
        if (e instanceof ApiError && e.status === 401) return null
        throw e
      }
    },
    staleTime: 5 * 60_000,
    retry: false,
  })
}

// Mutations change server state; afterwards we update the client cache so
// the UI reflects reality without a page reload.

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.login,
    // We already hold the fresh user — write it into the cache directly.
    onSuccess: (user) => qc.setQueryData(['me'], user),
  })
}

export function useRegister() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.register,
    onSuccess: (user) => qc.setQueryData(['me'], user),
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.logout,
    onSuccess: () => qc.setQueryData(['me'], null),
  })
}

export function useCreateCategory() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createCategory,
    // We changed the categories list on the server — mark every cached
    // copy stale so visible ones refetch.
    onSuccess: () => qc.invalidateQueries({ queryKey: ['categories'] }),
  })
}
