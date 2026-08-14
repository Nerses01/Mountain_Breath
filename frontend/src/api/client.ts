import type { Currency } from '../lib/currencies'
import type {
  Address,
  AdminProduct,
  CheckoutInput,
  AdminReview,
  ApiErrorBody,
  NewReview,
  Review,
  ReviewStatus,
  Cart,
  CatalogFacets,
  Category,
  Credentials,
  EditorialInput,
  ImageInput,
  NewCategory,
  NewProduct,
  Order,
  OrderStatus,
  Paginated,
  Product,
  ProductDetail,
  ProductSort,
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

/**
 * The language every request is made in.
 *
 * The API resolves it as ?lang= > mb_locale cookie > Accept-Language > en,
 * and until E2 the frontend used NONE of those — so a visitor on /hy/shop
 * got an Armenian header wrapped around an English catalog. The bug was
 * invisible to every test: the backend's fallback chain returns valid
 * English, so nothing failed, it was just wrong.
 *
 * Set from useLocale (one direction, like everything else about locale) and
 * appended to every request here rather than threaded through forty call
 * sites. It is deliberately NOT a per-call argument for writes: an admin
 * POST does not have a display language.
 *
 * The catch this pattern always has: TanStack Query would happily serve the
 * cached English response after a switch, because the URL changed but the
 * query KEY did not. So `apiLocale` also goes into every catalog query key
 * (see hooks.ts) — module state that changes what a request returns must be
 * part of that request's identity.
 */
let apiLocale = 'en'

export function setApiLocale(locale: string) {
  apiLocale = locale
}

/**
 * The market every request is made in — the same mechanism as apiLocale, and
 * with the same caveat about query keys (see hooks.ts).
 *
 * It is NOT only a display concern, which is the part that surprises: the
 * price filter's bounds, the price sort and the slider's ends are all
 * denominated in this currency server-side, so a request that forgets it
 * gets a correctly-shaped answer to the wrong question. And on POST /orders
 * it decides what the customer is charged in.
 */
let apiCurrency = 'USD'

export function setApiCurrency(currency: string) {
  apiCurrency = currency
}

function withView(path: string): string {
  const sep = path.includes('?') ? '&' : '?'
  return `${path}${sep}lang=${apiLocale}&currency=${apiCurrency}`
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

// CatalogFilterParams is everything BOTH catalog endpoints understand. The
// listing adds paging on top; the facets endpoint takes this and nothing
// more, because its counts describe the whole filtered catalog rather than
// one page.
export interface CatalogFilterParams {
  category?: string
  q?: string
  benefits?: string[]
  minPriceMinor?: number
  maxPriceMinor?: number
  sort?: ProductSort
}

export interface ProductListParams extends CatalogFilterParams {
  page?: number
  perPage?: number
}

// One place builds the shared half of the query string, so the sidebar and
// the grid can never disagree about what the current filter is.
function catalogQuery(params: CatalogFilterParams): URLSearchParams {
  const q = new URLSearchParams()
  if (params.category) q.set('category', params.category)
  if (params.q) q.set('q', params.q)
  // append, not set: the API reads repeated params, so several chips arrive
  // as ?benefit=energy&benefit=skin rather than one comma-joined value that
  // would need an escaping rule.
  for (const b of params.benefits ?? []) q.append('benefit', b)
  if (params.minPriceMinor !== undefined) q.set('min_price', String(params.minPriceMinor))
  if (params.maxPriceMinor !== undefined) q.set('max_price', String(params.maxPriceMinor))
  if (params.sort) q.set('sort', params.sort)
  return q
}

export const api = {
  // Every READ that returns human-language text carries the locale. Writes
  // and auth do not: a POST has no display language.
  listCategories: () => request<Category[]>(withView('/api/v1/categories')),

  listProducts: (params: ProductListParams = {}) => {
    const q = catalogQuery(params)
    if (params.page) q.set('page', String(params.page))
    if (params.perPage) q.set('per_page', String(params.perPage))
    const qs = q.toString()
    return request<Paginated<Product>>(withView(`/api/v1/products${qs ? `?${qs}` : ''}`))
  },

  catalogFacets: (params: CatalogFilterParams = {}) => {
    const qs = catalogQuery(params).toString()
    return request<CatalogFacets>(withView(`/api/v1/catalog/facets${qs ? `?${qs}` : ''}`))
  },

  getProduct: (slug: string) =>
    request<ProductDetail>(withView(`/api/v1/products/${encodeURIComponent(slug)}`)),

  // Its own request, not a field on the detail: the panel sits below the
  // fold and changes far less often than stock or price, so the two cache
  // under different keys.
  // `curated: true` asks for ONLY the admin's list, with no computed
  // fallback — the admin picker's pre-fill, and never the storefront's read.
  relatedProducts: (slug: string, curated = false) =>
    request<Product[]>(
      withView(
        `/api/v1/products/${encodeURIComponent(slug)}/related${curated ? '?curated=true' : ''}`,
      ),
    ),

  // --- Reviews (E4) ---
  //
  // The public list needs no `status` param and deliberately offers none:
  // the endpoint pins it to `published` server-side, so there is nothing a
  // client could ask for that would expose an unmoderated review.
  listReviews: (slug: string, page = 1) =>
    request<Paginated<Review>>(
      withView(`/api/v1/products/${encodeURIComponent(slug)}/reviews?page=${page}`),
    ),
  createReview: (slug: string, review: NewReview) =>
    request<Review>(`/api/v1/products/${encodeURIComponent(slug)}/reviews`, {
      method: 'POST',
      body: review,
    }),
  adminReviews: (status?: ReviewStatus) =>
    request<Paginated<AdminReview>>(
      `/api/v1/admin/reviews${status ? `?status=${status}` : ''}`,
    ),
  moderateReview: (id: number, status: ReviewStatus) =>
    request<Review>(`/api/v1/admin/reviews/${id}`, {
      method: 'PATCH',
      body: { status },
    }),

  // auth — the browser attaches the session cookie automatically
  me: () => request<User>('/api/v1/auth/me'),
  register: (creds: Credentials) =>
    request<User>('/api/v1/auth/register', { method: 'POST', body: creds }),
  login: (creds: Credentials) =>
    request<User>('/api/v1/auth/login', { method: 'POST', body: creds }),
  logout: () => request<void>('/api/v1/auth/logout', { method: 'POST' }),

  // cart & orders (require login). The cart carries product NAMES, so it is
  // a localized read like the catalog; order items are frozen snapshots of
  // what was bought and are not re-translated.
  getCart: () => request<Cart>(withView('/api/v1/cart')),
  setCartItem: (variantId: number, qty: number) =>
    // A write that answers with the whole cart, so the response is as
    // language- and currency-shaped as the read.
    request<Cart>(withView('/api/v1/cart/items'), {
      method: 'PUT',
      body: { variant_id: variantId, qty },
    }),
  removeCartItem: (variantId: number) =>
    request<void>(`/api/v1/cart/items/${variantId}`, { method: 'DELETE' }),
  // The currency in the URL is what the customer is BILLED in — the server
  // reads it from the request, never from a body field, so a client cannot
  // name the cheaper of two markets for a basket priced in the dearer one.
  // The body carries the customer's CHOICES and no money (E6).
  checkout: (input: CheckoutInput) =>
    request<Order>(withView('/api/v1/orders'), { method: 'POST', body: input }),
  myOrders: () => request<Order[]>('/api/v1/orders'),
  getOrder: (id: number) => request<Order>(`/api/v1/orders/${id}`),
  // 404 means "no saved address yet" — a first checkout — and the caller
  // renders an empty form for it rather than an error.
  defaultAddress: () => request<Address>('/api/v1/account/address'),

  // admin
  createCategory: (data: NewCategory) =>
    request<Category>('/api/v1/admin/categories', { method: 'POST', body: data }),
  adminOrders: () => request<Order[]>('/api/v1/admin/orders'),
  adminProducts: () =>
    request<Paginated<AdminProduct>>(withView('/api/v1/admin/products?per_page=100')),
  createProduct: (p: NewProduct) =>
    request<AdminProduct>('/api/v1/admin/products', { method: 'POST', body: p }),
  updateProduct: (id: number, p: UpdateProduct) =>
    request<AdminProduct>(`/api/v1/admin/products/${id}`, { method: 'PUT', body: p }),
  updateVariant: (id: number, prices: Partial<Record<Currency, number>>, stockQty: number) =>
    request<void>(`/api/v1/admin/variants/${id}`, {
      method: 'PATCH',
      // The DESIRED STATE of this variant's prices, not a patch: a currency
      // left out of the map is removed, which is how the admin puts a variant
      // back on the converted fallback.
      body: { prices, stock_qty: stockQty },
    }),
  // E3 editorial writes. All 204 No Content — the form already holds the
  // state it just sent, so echoing it back would only invite the two to
  // disagree.
  saveProductImages: (id: number, images: ImageInput[]) =>
    request<void>(`/api/v1/admin/products/${id}/images`, {
      method: 'PUT',
      body: { images },
    }),
  deleteProductImage: (productId: number, imageId: number) =>
    request<void>(`/api/v1/admin/products/${productId}/images/${imageId}`, {
      method: 'DELETE',
    }),
  saveProductEditorial: (id: number, content: Record<string, EditorialInput>) =>
    request<void>(`/api/v1/admin/products/${id}/editorial`, {
      method: 'PUT',
      body: { content },
    }),
  saveProductRelated: (id: number, relatedIds: number[]) =>
    request<void>(`/api/v1/admin/products/${id}/related`, {
      method: 'PUT',
      body: { related_ids: relatedIds },
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
