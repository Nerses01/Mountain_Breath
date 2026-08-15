import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { api, ApiError, type CatalogFilterParams, type ProductListParams } from './client'
import type {
  Address,
  AddressInput,
  EditorialInput,
  Money,
  ImageInput,
  NewReview,
  OrderStatus,
  ReviewStatus,
  UpdateProduct,
  User,
} from './types'
import { useLocale } from '../i18n/useLocale'
import { useCurrency } from '../lib/useCurrency'

// TanStack Query caches by queryKey: two components asking for the same key
// share one request and one cached result.
//
// Every key that fetches TRANSLATED text carries the locale. Without it a
// language switch would change the request URL but not the cache key, so the
// cached English response would be served for the Armanian page and only a
// manual refetch would fix it — a caching bug that looks exactly like a
// translation bug.
//
// E5 adds the currency to every key that fetches a PRICE, for exactly the
// same reason and with exactly the same failure mode: switching to drams
// would change the URL and not the key, so the dollar prices would stay on
// screen until something else happened to refetch. Module state that changes
// what a request returns must be part of that request's identity — twice
// over now.
//
// `view` is the pair, named once so a key cannot accidentally carry one and
// not the other.
function useView(): [string, string] {
  const { locale } = useLocale()
  const { currency } = useCurrency()
  return [locale, currency]
}

export function useCategories() {
  const { locale } = useLocale()
  return useQuery({
    queryKey: ['categories', locale],
    queryFn: api.listCategories,
  })
}

// `enabled: false` holds the request back without unmounting the caller —
// the search overlay uses it to stay quiet until the term is worth a round
// trip, the same lever useCart pulls for anonymous visitors.
export function useProducts(params: ProductListParams, enabled = true) {
  const view = useView()
  return useQuery({
    // params are part of the key — changing the category filter is a new
    // cache entry, and going back to a seen filter is instant.
    queryKey: ['products', ...view, params],
    queryFn: () => api.listProducts(params),
    enabled,
    // Keep the previous page on screen while the next one loads, instead of
    // blanking the grid on every filter click. The sidebar stays put, the
    // products dim — which is what "loading" should look like once there is
    // already something to look at.
    placeholderData: keepPreviousData,
  })
}

// The sidebar's counts. A SEPARATE query from useProducts, deliberately:
// its key omits page and per_page, so paging through results reuses the
// cached facets instead of re-running the expensive counting query.
export function useCatalogFacets(params: CatalogFilterParams) {
  const view = useView()
  return useQuery({
    // The currency is in here twice over: it changes the counts' price
    // bounds AND the meaning of the min/max already inside `params`.
    queryKey: ['catalog-facets', ...view, params],
    queryFn: () => api.catalogFacets(params),
    placeholderData: keepPreviousData,
  })
}

export function useProduct(slug: string) {
  const view = useView()
  return useQuery({
    queryKey: ['product', ...view, slug],
    queryFn: () => api.getProduct(slug),
  })
}

// "Often taken together". A separate key from ['product', …] so scrolling
// past it does not invalidate the buy box, and so a stock change that
// refetches the product leaves this panel alone.
export function useRelatedProducts(slug: string, curated = false) {
  const view = useView()
  return useQuery({
    // `curated` is part of the key: the two answers are different lists, and
    // sharing a key would let the storefront's computed panel be served to
    // the admin picker, which is the exact confusion the flag exists to
    // prevent.
    queryKey: ['related', ...view, slug, curated],
    queryFn: () => api.relatedProducts(slug, curated),
  })
}

// --- Reviews (E4) ---

export function useReviews(slug: string, page = 1) {
  const { locale } = useLocale()
  return useQuery({
    queryKey: ['reviews', locale, slug, page],
    queryFn: () => api.listReviews(slug, page),
    placeholderData: keepPreviousData,
  })
}

export function useCreateReview(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (review: NewReview) => api.createReview(slug, review),
    onSuccess: () => {
      // The review lands PENDING, so the public list does not change — but
      // `can_review` on the product just flipped to false, and the form must
      // disappear. That is the detail query, not the review list.
      qc.invalidateQueries({ queryKey: ['product'] })
      qc.invalidateQueries({ queryKey: ['reviews'] })
    },
  })
}

export function useAdminReviews(status?: ReviewStatus) {
  return useQuery({
    queryKey: ['admin-reviews', status ?? 'all'],
    queryFn: () => api.adminReviews(status),
  })
}

export function useModerateReview() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: number; status: ReviewStatus }) =>
      api.moderateReview(id, status),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-reviews'] })
      // Publishing or rejecting moves the product's stored average, and the
      // average is on the CARD as well as the detail — so both catalog
      // caches are stale, not just the one product.
      qc.invalidateQueries({ queryKey: ['reviews'] })
      qc.invalidateQueries({ queryKey: ['product'] })
      qc.invalidateQueries({ queryKey: ['products'] })
    },
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

// enabled: only fetch the cart when someone is logged in — an anonymous
// visitor would just collect 401s.
export function useCart(loggedIn: boolean) {
  const view = useView()
  return useQuery({
    // The cart carries product names AND prices, so it is a translated read
    // and a priced one.
    queryKey: ['cart', ...view],
    queryFn: api.getCart,
    enabled: loggedIn,
  })
}

export function useSetCartItem() {
  const qc = useQueryClient()
  const view = useView()
  return useMutation({
    mutationFn: ({ variantId, qty }: { variantId: number; qty: number }) =>
      api.setCartItem(variantId, qty),
    // The response IS the updated cart — write it straight into the cache.
    // The key must match useCart's EXACTLY, view and all: setQueryData is an
    // exact-key write, unlike invalidateQueries, which matches by prefix. A
    // stale ['cart'] here would silently stop updating the header count.
    onSuccess: (cart) => {
      qc.setQueryData(['cart', ...view], cart)
      // Every quantity change moves the preview's money — subtotal,
      // discounts, whether the free-shipping bar is full.
      qc.invalidateQueries({ queryKey: ['preview'] })
    },
  })
}

export function useRemoveCartItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.removeCartItem,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cart'] })
      qc.invalidateQueries({ queryKey: ['preview'] })
    },
  })
}

// --- Checkout preview & promos (E7) ---

// The one calculator's client end: every money figure the cart page and the
// checkout sidebar render comes from this query. Keyed by the view — the
// preview is a priced, translated read like the cart itself.
export function usePreview(loggedIn: boolean) {
  const view = useView()
  return useQuery({
    queryKey: ['preview', ...view],
    queryFn: api.checkoutPreview,
    enabled: loggedIn,
  })
}

// Both promo mutations answer with a fresh preview, and both write it
// straight into the cache under the exact view key (the setCartItem
// pattern) — no second round trip to learn what just changed.
export function useApplyPromo() {
  const qc = useQueryClient()
  const view = useView()
  return useMutation({
    mutationFn: api.applyPromo,
    onSuccess: (preview) => qc.setQueryData(['preview', ...view], preview),
  })
}

export function useRemovePromo() {
  const qc = useQueryClient()
  const view = useView()
  return useMutation({
    mutationFn: api.removePromo,
    onSuccess: (preview) => qc.setQueryData(['preview', ...view], preview),
  })
}

export function useCheckout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.checkout,
    onSuccess: () => {
      // Checkout changes a lot of server state: cart emptied, order added,
      // stock decremented (affects product lists and detail pages) — and
      // since E6 the address book gained the form's address, which is what
      // the NEXT checkout pre-fills from.
      qc.invalidateQueries({ queryKey: ['cart'] })
      qc.invalidateQueries({ queryKey: ['preview'] })
      qc.invalidateQueries({ queryKey: ['orders'] })
      qc.invalidateQueries({ queryKey: ['products'] })
      qc.invalidateQueries({ queryKey: ['product'] })
      qc.invalidateQueries({ queryKey: ['default-address'] })
      // E7: the order that just landed may have been the FIRST — the hive
      // standing on /auth/me (header badge, perk lines) just changed.
      qc.invalidateQueries({ queryKey: ['me'] })
    },
  })
}

export function useMyOrders() {
  return useQuery({
    queryKey: ['orders'],
    queryFn: api.myOrders,
  })
}

// The confirmation page's read. No locale or currency in the key: an order
// is a frozen record — its snapshots do not change with the viewer's
// language or market, so one cache entry serves every view of it.
export function useOrder(id: number) {
  return useQuery({
    queryKey: ['orders', id],
    queryFn: () => api.getOrder(id),
    enabled: Number.isFinite(id) && id > 0,
  })
}

// --- Account (E8) ---

// The wishlist doubles as the heart-state oracle: every heart on every card
// derives its on/off from this one query's product ids — six products make
// membership a Set lookup, not a per-product endpoint.
export function useWishlist(loggedIn: boolean) {
  const view = useView()
  return useQuery({
    queryKey: ['wishlist', ...view],
    queryFn: api.getWishlist,
    enabled: loggedIn,
  })
}

export function useToggleWishlist() {
  const qc = useQueryClient()
  return useMutation({
    // One mutation for both directions — the caller states the DESIRED
    // state, mirroring the API's set-semantics.
    mutationFn: ({ productId, hearted }: { productId: number; hearted: boolean }) =>
      hearted ? api.addWishlistItem(productId) : api.removeWishlistItem(productId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['wishlist'] }),
  })
}

export function useSaveForLater() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.saveForLater,
    onSuccess: () => {
      // A line left the cart and a heart appeared — three caches moved.
      qc.invalidateQueries({ queryKey: ['cart'] })
      qc.invalidateQueries({ queryKey: ['preview'] })
      qc.invalidateQueries({ queryKey: ['wishlist'] })
    },
  })
}

export function useForgotPassword() {
  return useMutation({ mutationFn: api.forgotPassword })
}

// --- Newsletter (E9) --- no query cache involved: three fire-and-forget
// mutations whose whole state lives in the mutation object itself.

export function useSubscribeNewsletter() {
  return useMutation({ mutationFn: api.subscribeNewsletter })
}

export function useConfirmNewsletter() {
  return useMutation({ mutationFn: api.confirmNewsletter })
}

export function useUnsubscribeNewsletter() {
  return useMutation({ mutationFn: api.unsubscribeNewsletter })
}

export function useResetPassword() {
  return useMutation({
    mutationFn: ({ token, password }: { token: string; password: string }) =>
      api.resetPassword(token, password),
  })
}

export function useAddresses(enabled: boolean) {
  return useQuery({
    queryKey: ['addresses'],
    queryFn: api.listAddresses,
    enabled,
  })
}

// All three writes invalidate the book AND the checkout's prefill — the
// default address is the same fact read through two endpoints.
function useInvalidateAddresses() {
  const qc = useQueryClient()
  return () => {
    qc.invalidateQueries({ queryKey: ['addresses'] })
    qc.invalidateQueries({ queryKey: ['default-address'] })
  }
}

export function useCreateAddress() {
  const invalidate = useInvalidateAddresses()
  return useMutation({ mutationFn: api.createAddress, onSuccess: invalidate })
}

export function useUpdateAddress() {
  const invalidate = useInvalidateAddresses()
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: AddressInput }) =>
      api.updateAddress(id, input),
    onSuccess: invalidate,
  })
}

export function useDeleteAddress() {
  const invalidate = useInvalidateAddresses()
  return useMutation({ mutationFn: api.deleteAddress, onSuccess: invalidate })
}

// The saved address for pre-filling the checkout form. 404 (first-time
// customer) resolves to null — an empty form is a normal state, not an error,
// the same mapping useMe applies to 401.
export function useDefaultAddress(enabled: boolean) {
  return useQuery<Address | null>({
    queryKey: ['default-address'],
    queryFn: async () => {
      try {
        return await api.defaultAddress()
      } catch (e) {
        if (e instanceof ApiError && e.status === 404) return null
        throw e
      }
    },
    enabled,
  })
}

export function useAdminOrders() {
  return useQuery({
    queryKey: ['admin-orders'],
    queryFn: api.adminOrders,
  })
}

export function useAdminProducts() {
  const view = useView()
  return useQuery({
    // The admin list shows prices too, and its editor writes back what it
    // reads — a cached dollar figure shown under a dram heading would be
    // saved as a dram price on the next keystroke.
    queryKey: ['admin-products', ...view],
    queryFn: api.adminProducts,
  })
}

// All product writes invalidate both worlds: the admin list and the public
// catalog caches (lists + detail pages).
function useInvalidateProducts() {
  const qc = useQueryClient()
  return () => {
    qc.invalidateQueries({ queryKey: ['admin-products'] })
    qc.invalidateQueries({ queryKey: ['products'] })
    qc.invalidateQueries({ queryKey: ['product'] })
    // Adding, hiding or recategorising a product moves the sidebar counts —
    // easy to forget precisely because nothing on the admin screen shows them.
    qc.invalidateQueries({ queryKey: ['catalog-facets'] })
    // E3: curating a related list, or editing a product at all, changes what
    // other products' "Often taken together" panels show — the fallback
    // ranks by shared benefits and popularity, both of which product edits
    // can move.
    qc.invalidateQueries({ queryKey: ['related'] })
  }
}

export function useCreateProduct() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: api.createProduct,
    onSuccess: invalidate,
  })
}

export function useUpdateProduct() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateProduct }) =>
      api.updateProduct(id, data),
    onSuccess: invalidate,
  })
}

export function useUploadProductImage() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: ({ id, file }: { id: number; file: File }) =>
      api.uploadProductImage(id, file),
    onSuccess: invalidate,
  })
}

// E3 editorial writes. All four invalidate the same caches as any other
// product edit — a reordered gallery or a new bullet changes the product
// page, and a curated related list changes another product's panel.
export function useSaveProductImages() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: ({ id, images }: { id: number; images: ImageInput[] }) =>
      api.saveProductImages(id, images),
    onSuccess: invalidate,
  })
}

export function useDeleteProductImage() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: ({ productId, imageId }: { productId: number; imageId: number }) =>
      api.deleteProductImage(productId, imageId),
    onSuccess: invalidate,
  })
}

export function useSaveProductEditorial() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: ({ id, content }: { id: number; content: Record<string, EditorialInput> }) =>
      api.saveProductEditorial(id, content),
    onSuccess: invalidate,
  })
}

export function useSaveProductRelated() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: ({ id, relatedIds }: { id: number; relatedIds: number[] }) =>
      api.saveProductRelated(id, relatedIds),
    // Also drops the cached "Often taken together" of every product, since
    // curating one list can change what another product's fallback returns.
    onSuccess: () => invalidate(),
  })
}

export function useUpdateVariant() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: ({ id, prices, stockQty }: { id: number; prices: Money; stockQty: number }) =>
      api.updateVariant(id, prices, stockQty),
    onSuccess: invalidate,
  })
}

export function useUpdateOrderStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orderId, status }: { orderId: number; status: OrderStatus }) =>
      api.updateOrderStatus(orderId, status),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-orders'] })
      qc.invalidateQueries({ queryKey: ['orders'] })
      // cancelling restores stock
      qc.invalidateQueries({ queryKey: ['products'] })
      qc.invalidateQueries({ queryKey: ['product'] })
    },
  })
}
