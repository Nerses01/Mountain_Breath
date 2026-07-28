import type { ApiErrorBody, Category, Paginated, Product } from './types'

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly fields?: Record<string, string>

  constructor(
    status: number,
    code: string,
    message: string,
    fields?: Record<string, string>,
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
  }
}

async function request<T>(path: string): Promise<T> {
  const res = await fetch(path)
  if (!res.ok) {
    // Try to read our standard error envelope; fall back to the HTTP status.
    let code = 'unknown_error'
    let message = `HTTP ${res.status}`
    let fields: Record<string, string> | undefined
    try {
      const body = (await res.json()) as ApiErrorBody
      code = body.error.code
      message = body.error.message
      fields = body.error.fields
    } catch {
      // response body wasn't our JSON envelope — keep the fallback
    }
    throw new ApiError(res.status, code, message, fields)
  }
  return res.json() as Promise<T>
}

export interface ProductListParams {
  category?: string
  page?: number
  perPage?: number
}

export const api = {
  listCategories: () => request<Category[]>('/api/v1/categories'),

  listProducts: (params: ProductListParams = {}) => {
    const q = new URLSearchParams()
    if (params.category) q.set('category', params.category)
    if (params.page) q.set('page', String(params.page))
    if (params.perPage) q.set('per_page', String(params.perPage))
    const qs = q.toString()
    return request<Paginated<Product>>(`/api/v1/products${qs ? `?${qs}` : ''}`)
  },

  getProduct: (slug: string) =>
    request<Product>(`/api/v1/products/${encodeURIComponent(slug)}`),
}
