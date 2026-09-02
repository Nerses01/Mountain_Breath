import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import { api, ApiError, type CatalogFilterParams, type ProductListParams } from './client'
import type {
  AddressEntry,
  AddressInput,
  EditorialInput,
  Money,
  ImageInput,
  NewCategory,
  NewReview,
  OrderStatus,
  PaymentStatus,
  PromoInput,
  Role,
  Product,
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

// Category writes stale BOTH lists: the storefront's locale-resolved
// ['categories'] and the editor's raw ['admin-categories'].
function useInvalidateCategories() {
  const qc = useQueryClient()
  return () => {
    qc.invalidateQueries({ queryKey: ['categories'] })
    qc.invalidateQueries({ queryKey: ['admin-categories'] })
  }
}

export function useCreateCategory() {
  const invalidate = useInvalidateCategories()
  return useMutation({
    mutationFn: api.createCategory,
    onSuccess: invalidate,
  })
}

// --- F2 category management (decision #95) -------------------------------

export function useAdminCategories() {
  return useQuery({
    queryKey: ['admin-categories'],
    queryFn: api.adminCategories,
  })
}

export function useUpdateCategory() {
  const invalidate = useInvalidateCategories()
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: NewCategory }) =>
      api.updateCategory(id, input),
    // A renamed or re-slugged category also changes product payloads
    // (their category object) and the catalog facets.
    onSuccess: () => {
      invalidate()
    },
  })
}

export function useDeleteCategory() {
  const invalidate = useInvalidateCategories()
  return useMutation({
    mutationFn: api.deleteCategory,
    onSuccess: invalidate,
  })
}

export function useReorderCategories() {
  const invalidate = useInvalidateCategories()
  return useMutation({
    mutationFn: api.reorderCategories,
    onSuccess: invalidate,
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

/**
 * The product CARD's "Add to cart", shared by the home page, the shop grid
 * and the wishlist. One contract in one place: the cheapest in-stock
 * variant — the price the card is showing — one more than whatever the cart
 * already holds.
 *
 * "+1", never "set to 1": setCartItem is a PUT of an absolute quantity, and
 * three copies of this handler once disagreed on that — two sent qty 1, so
 * every click after the first silently re-set the same quantity and the
 * button looked dead. Returns undefined for a signed-out visitor, which the
 * card renders as a disabled button.
 *
 * The promise resolves to how many of this product the cart now holds —
 * read from the RESPONSE, not assumed from what was sent, so the button's
 * "In cart: 2" flash can never show a number the server did not confirm.
 */
export function useQuickAdd(): ((product: Product) => Promise<number>) | undefined {
  const me = useMe()
  const cart = useCart(!!me.data)
  const setCartItem = useSetCartItem()

  if (!me.data) return undefined
  return async (product) => {
    const variant = product.variants.find((v) => v.stock_qty > 0)
    if (!variant) return 0
    const inCart = cart.data?.items.find((it) => it.variant_id === variant.id)?.qty ?? 0
    // At the ceiling: no request — the cart already holds every unit the
    // shop has. Returning the held count still flashes "In cart: N", which
    // is the honest answer to the click ("you have them all"), instead of
    // silently re-setting the same quantity — the dead-button bug's shape.
    if (inCart >= variant.stock_qty) return inCart
    const updated = await setCartItem.mutateAsync({ variantId: variant.id, qty: inCart + 1 })
    return updated.items.find((it) => it.variant_id === variant.id)?.qty ?? inCart + 1
  }
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

/**
 * A2: refill the cart from a past order. The server merges in one
 * transaction and reports each line's fate; success here only means the
 * MERGE ran — the caller reads result.lines to tell the customer what was
 * added, capped, or skipped. Cart and preview caches are stale either way.
 */
export function useReorder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.reorder,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cart'] })
      qc.invalidateQueries({ queryKey: ['preview'] })
    },
  })
}

// --- Settings (A5) ------------------------------------------------------

/** The profile PATCH echoes the fresh user, so the ['me'] cache is SET
 *  from the response — no refetch round-trip for data we just received. */
export function useUpdateProfile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.updateProfile,
    onSuccess: (user) => qc.setQueryData(['me'], user),
  })
}

export function useChangePassword() {
  return useMutation({ mutationFn: api.changePassword })
}

export function useNotifications(enabled: boolean) {
  return useQuery({
    queryKey: ['notifications'],
    queryFn: api.getNotifications,
    enabled,
  })
}

export function useSetOrderUpdates() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.setOrderUpdates,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['notifications'] }),
  })
}

/** The harvest-notes toggle: ON re-runs the newsletter's own double
 *  opt-in (consent stays verified), OFF unsubscribes by the session's
 *  email. Both leave the panel's query stale. */
export function useNewsletterToggle() {
  const qc = useQueryClient()
  const invalidate = () => qc.invalidateQueries({ queryKey: ['notifications'] })
  const subscribe = useMutation({
    mutationFn: api.subscribeNewsletter,
    onSuccess: invalidate,
  })
  const unsubscribe = useMutation({
    mutationFn: api.accountUnsubscribeNewsletter,
    onSuccess: invalidate,
  })
  return { subscribe, unsubscribe }
}

/** A3: the wishlist's "Add all to cart" — same contract as useReorder. */
export function useAddWishlistToCart() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.addWishlistToCart,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['cart'] })
      qc.invalidateQueries({ queryKey: ['preview'] })
    },
  })
}

// The confirmation page's read. No locale or currency in the key: an order
// is a frozen record — its snapshots do not change with the viewer's
// language or market, so one cache entry serves every view of it.
// F2: cancel a pending order. Invalidation mirrors the admin's status
// mutation — cancelling restores stock, so the product caches go too. On
// a 409 (the shop confirmed while the page was open) the ORDER caches are
// also dropped: the freshest thing to show under the error is the real
// status that made the cancel too late.
export function useCancelOrder() {
  const qc = useQueryClient()
  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['orders'] })
    qc.invalidateQueries({ queryKey: ['products'] })
    qc.invalidateQueries({ queryKey: ['product'] })
  }
  return useMutation({
    mutationFn: api.cancelOrder,
    onSuccess: invalidate,
    onError: (e) => {
      if (e instanceof ApiError && e.status === 409) {
        qc.invalidateQueries({ queryKey: ['orders'] })
      }
    },
  })
}

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
  return useQuery<AddressEntry | null>({
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

export function useUploadProductVideo() {
  const invalidate = useInvalidateProducts()
  return useMutation({
    mutationFn: ({ id, file }: { id: number; file: File }) =>
      api.uploadProductVideo(id, file),
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

// F2 (decision #97): the point of no return. On success the WHOLE cache
// is cleared, not invalidated — every cached fact is about an account
// that no longer exists, and a refetch would only collect 401s.
export function useDeleteAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteAccount,
    onSuccess: () => {
      qc.clear()
    },
  })
}

// F2 (decision #96): user administration. A role change also invalidates
// ['me'] — an admin demoting THEMSELF (with another admin left) must see
// the admin area close, not linger on a cached role.
export function useAdminUsers() {
  return useQuery({
    queryKey: ['admin-users'],
    queryFn: api.adminUsers,
  })
}

export function useUpdateUserRole() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, role }: { id: number; role: Role }) =>
      api.updateUserRole(id, role),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-users'] })
      qc.invalidateQueries({ queryKey: ['me'] })
    },
  })
}

// F2 (decision #94): the admin's promo CRUD. Writes also drop the cart
// preview cache — an edited code can change what an open cart's applied
// promo is worth, and the preview re-judges on every read by design.
export function useAdminPromos() {
  return useQuery({
    queryKey: ['admin-promos'],
    queryFn: api.adminPromos,
  })
}

function useInvalidatePromos() {
  const qc = useQueryClient()
  return () => {
    qc.invalidateQueries({ queryKey: ['admin-promos'] })
    qc.invalidateQueries({ queryKey: ['preview'] })
  }
}

export function useCreatePromo() {
  const invalidate = useInvalidatePromos()
  return useMutation({ mutationFn: api.createPromo, onSuccess: invalidate })
}

export function useUpdatePromo() {
  const invalidate = useInvalidatePromos()
  return useMutation({
    mutationFn: ({ id, input }: { id: number; input: PromoInput }) =>
      api.updatePromo(id, input),
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

// F2: flip an order's payment status (mark paid / refunded). Narrower
// invalidation than its status sibling: a payment flip never touches
// stock, so the product caches stay put — only the two order views
// (the admin table and the customer's own pages) need a refetch.
export function useUpdateOrderPayment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ orderId, paymentStatus }: { orderId: number; paymentStatus: PaymentStatus }) =>
      api.updateOrderPayment(orderId, paymentStatus),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin-orders'] })
      qc.invalidateQueries({ queryKey: ['orders'] })
    },
  })
}
