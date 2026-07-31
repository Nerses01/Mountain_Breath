import type {
  AdminProduct,
  ApiErrorBody,
  Cart,
  Category,
  Credentials,
  NewCategory,
  NewProduct,
  Order,
  OrderStatus,
  Paginated,
  Product,
  UpdateProduct,
  User,
} from './types'

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

interface RequestOptions {
  method?: string
  body?: unknown
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  // FormData goes through untouched — the browser sets the multipart
  // Content-Type (with boundary) itself; setting it manually breaks uploads.
  const isForm = options.body instanceof FormData
  const res = await fetch(path, {
    method: options.method ?? 'GET',
    headers:
      options.body !== undefined && !isForm
        ? { 'Content-Type': 'application/json' }
        : undefined,
    body: isForm
      ? (options.body as FormData)
      : options.body !== undefined
        ? JSON.stringify(options.body)
        : undefined,
  })
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
  if (res.status === 204) {
    return undefined as T // no content (e.g. logout)
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

  // auth — the browser attaches the session cookie automatically
  me: () => request<User>('/api/v1/auth/me'),
  register: (creds: Credentials) =>
    request<User>('/api/v1/auth/register', { method: 'POST', body: creds }),
  login: (creds: Credentials) =>
    request<User>('/api/v1/auth/login', { method: 'POST', body: creds }),
  logout: () => request<void>('/api/v1/auth/logout', { method: 'POST' }),

  // cart & orders (require login)
  getCart: () => request<Cart>('/api/v1/cart'),
  setCartItem: (variantId: number, qty: number) =>
    request<Cart>('/api/v1/cart/items', {
      method: 'PUT',
      body: { variant_id: variantId, qty },
    }),
  removeCartItem: (variantId: number) =>
    request<void>(`/api/v1/cart/items/${variantId}`, { method: 'DELETE' }),
  checkout: () => request<Order>('/api/v1/orders', { method: 'POST' }),
  myOrders: () => request<Order[]>('/api/v1/orders'),

  // admin
  createCategory: (data: NewCategory) =>
    request<Category>('/api/v1/admin/categories', { method: 'POST', body: data }),
  adminOrders: () => request<Order[]>('/api/v1/admin/orders'),
  adminProducts: () =>
    request<Paginated<AdminProduct>>('/api/v1/admin/products?per_page=100'),
  createProduct: (p: NewProduct) =>
    request<AdminProduct>('/api/v1/admin/products', { method: 'POST', body: p }),
  updateProduct: (id: number, p: UpdateProduct) =>
    request<AdminProduct>(`/api/v1/admin/products/${id}`, { method: 'PUT', body: p }),
  updateVariant: (id: number, priceMinor: number, stockQty: number) =>
    request<void>(`/api/v1/admin/variants/${id}`, {
      method: 'PATCH',
      body: { price_minor: priceMinor, stock_qty: stockQty },
    }),
  uploadProductImage: (id: number, file: File) => {
    const form = new FormData()
    form.append('image', file)
    return request<{ image_url: string }>(`/api/v1/admin/products/${id}/image`, {
      method: 'POST',
      body: form,
    })
  },
  updateOrderStatus: (orderId: number, status: OrderStatus) =>
    request<Order>(`/api/v1/admin/orders/${orderId}/status`, {
      method: 'PATCH',
      body: { status },
    }),
}
