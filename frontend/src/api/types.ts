// Wire types: these mirror the Go DTOs in backend/internal/api (snake_case
// fields, exactly as JSON arrives). If the backend contract changes, change
// it here too — the compiler then points at every affected component.

import type { Currency } from '../lib/currencies'

/**
 * An amount in every market the shop could price it in — the wire shape of
 * Go's domain.Money.
 *
 * Partial, and that is load-bearing: a variant priced only in dollars, with
 * no exchange rate on file, arrives with one entry. Anything reading a
 * currency out of this must cope with `undefined`, which is exactly the
 * discipline a plain `Record<Currency, number>` would remove.
 */
export type Money = Partial<Record<Currency, number>>

export interface Category {
  id: number
  slug: string
  name: string
  sort_order: number
  created_at: string
}

export interface ProductVariant {
  id: number
  sku: string
  label: string
  stock_qty: number
  /**
   * The price in the response's `currency`, in THAT currency's minor units —
   * 1400 is $14.00 but 6700 is 6,700 ֏. Nothing may divide this by 100; use
   * formatMoney, which takes the scale from the currency.
   */
  price_minor: number
  /** The same variant in every market, for the design's muted second line. */
  prices: Money
}

// Badge keys the backend's CHECK constraint allows (migration 000009). A
// union, not `string`, so a typo in a translation lookup is a compile error
// and adding a badge to the database without adding its three translations
// fails the build rather than rendering a raw key on a card.
export type BadgeKey =
  | 'best_seller'
  | 'new'
  | 'cold_chain'
  | 'for_makers'
  | 'immunity'
  | 'protein'

export interface Benefit {
  slug: string
  name: string
}

export interface Product {
  id: number
  category_id: number
  // The category, already resolved into the requested language — the card's
  // eyebrow shows the name, the sidebar links on the slug.
  category_slug: string
  category_name: string
  /** The denormalized review aggregate — on the CARD as well as the detail,
   *  because the design draws stars in the grid too. */
  rating_avg: number
  rating_count: number
  slug: string
  name: string
  description: string
  image_url: string
  created_at: string
  variants: ProductVariant[]
  // '' when the product has no badge — the backend sends an empty string
  // rather than null so no consumer has to handle two kinds of absence.
  badge: BadgeKey | ''
  badge_tone: 'honey' | 'dark' | 'outline'
  benefits: Benefit[]
  /** What every variant's `price_minor` on this product is denominated in. */
  currency: Currency
}

// --- Reviews (E4) -------------------------------------------------------

export type ReviewStatus = 'pending' | 'published' | 'rejected'

export interface Review {
  id: number
  rating: number
  title: string
  body: string
  /** A display name derived from the email — never the address itself. */
  author: string
  created_at: string
  /** Present on the admin queue and on a just-created review; absent on the
   *  public list, where everything is published by definition. */
  status?: ReviewStatus
}

/** The moderation queue sees two things the storefront must not. */
export interface AdminReview extends Review {
  product_id: number
  email: string
}

export interface NewReview {
  rating: number
  title: string
  body: string
}

// --- Product detail (E3) ------------------------------------------------
//
// A separate interface EXTENDING Product rather than optional fields on it.
// The listing and the detail endpoint answer different questions, and one
// shared type would make every card in the grid carry six `| undefined`
// fields that a reader has to guess the population rules for.

export interface ProductImage {
  id: number
  url: string
  alt: string
  is_primary: boolean
}

export interface ProductHighlight {
  text: string
}

export interface ProductUsageCard {
  kicker: string
  title: string
  body: string
}

export interface ProductDetail extends Product {
  images: ProductImage[]
  highlights: ProductHighlight[]
  usage_cards: ProductUsageCard[]

  disclaimer: string
  storage_note: string
  harvest_note: string
  shipping_note: string
  lab_batch: string
  is_cold_chain: boolean
  /** Whether the CURRENT viewer may leave a review: signed in, has a
   *  delivered order for it, and has not reviewed it already. A hint for
   *  rendering — the write path checks the rule again. */
  can_review: boolean
}

// One row of a filter group: what to show, what to put in the URL, and how
// many products survive if it is clicked.
export interface FacetCount {
  slug: string
  name: string
  count: number
}

export interface CatalogFacets {
  categories: FacetCount[]
  benefits: FacetCount[]
  // The "All hive products" row: the total with the CATEGORY filter lifted.
  total: number
  price_min_minor: number
  price_max_minor: number
}

// The four orderings the backend whitelists. Keeping this a union means the
// sort select cannot offer a value the API would silently ignore.
export type ProductSort = 'popular' | 'rating' | 'price_asc' | 'price_desc' | 'newest'

export interface Paginated<T> {
  items: T[]
  page: number
  per_page: number
  total: number
}

export interface User {
  id: number
  email: string
  role: 'customer' | 'admin'
  /**
   * E7: hive-club standing, derived by the SERVER from order history — the
   * client renders these booleans, it never re-implements "after the first
   * order".
   */
  hive: {
    prior_orders: number
    member: boolean
    member_discount_percent: number
    first_delivery_free: boolean
  }
}

export interface Credentials {
  email: string
  password: string
}

/** Login is credentials plus the "keep me signed in" checkbox (E8): a week
 *  of session by default, thirty days when remembered. */
export interface LoginInput extends Credentials {
  remember: boolean
}

// --- Account (E8) -------------------------------------------------------

/** One row of the account page's address book: an Address with a name tag
 *  and the default flag the checkout prefills from. */
export interface AddressEntry extends Address {
  id: number
  label: string
  is_default: boolean
}

/** What the book's forms send — the id lives in the URL on updates. */
export interface AddressInput extends Address {
  label: string
  is_default: boolean
}

// Per-locale text. `name`/`description` on the parent objects are the ENGLISH
// copy and stay required — every other language falls back to them — so this
// map carries only hy and ru. An "en" key is rejected by the API.
export interface ProductText {
  name: string
  description: string
}

export interface NewCategory {
  slug: string
  name: string
  sort_order: number
  translations?: Record<string, { name: string }>
}

export interface CartItem {
  variant_id: number
  product_name: string
  product_slug: string
  label: string
  stock_qty: number
  qty: number
  /** Both denominated in the cart's `currency`. */
  price_minor: number
  line_total_minor: number
  /** The same line in every market. */
  prices: Money
  line_totals: Money
}

export interface Cart {
  items: CartItem[]
  currency: Currency
  /**
   * E7 removed shipping and totals from the cart: what delivery costs now
   * depends on WHO is asking (the hive club waives a first order's base) and
   * what code the cart carries, so every summary figure comes from
   * POST /checkout/preview instead. The cart is the list of lines; what
   * remains here is what the contents alone determine.
   */
  subtotal_minor: number
  /** True when any line is chilled — the "Chilled shipping" label's flag. */
  has_cold_chain: boolean
  /**
   * The sum-of-lines in every market, summed independently — never converted
   * from one to another, because rounding a sum is not the sum of roundings.
   * A market any line cannot be priced in is absent rather than understated.
   */
  subtotals: Money
}

// --- Checkout preview (E7) ----------------------------------------------

export interface Upsell {
  /** What the banner's button adds — a cart line is a variant. */
  variant_id: number
  slug: string
  name: string
  price_minor: number
}

/**
 * The one calculator's answer: every figure the cart page and the checkout
 * sidebar draw. The client's remaining arithmetic is formatting (and the
 * progress bar's width, which is display math, not money math).
 */
export interface Preview {
  currency: Currency

  subtotal_minor: number
  shipping_minor: number
  member_discount_minor: number
  promo_discount_minor: number
  discount_minor: number
  /** Contained in the subtotal ("Prices include VAT") — display, never add. */
  tax_minor: number
  total_minor: number

  has_cold_chain: boolean
  /** The hive-club perk: this is the customer's first order, base waived. */
  first_delivery_free: boolean
  /** The base is waived for ANY reason (perk, threshold, free-ship promo). */
  base_shipping_waived: boolean

  /** Both present only while the bar has something to count toward. */
  free_shipping_threshold_minor?: number
  free_shipping_remaining_minor?: number
  /** One product that would close the gap — the banner's CTA. */
  upsell?: Upsell

  /** The attached code, even when it cannot currently apply… */
  promo_code?: string
  promo_kind?: 'percent' | 'fixed' | 'free_shipping'
  /** …and the validation code saying why not ('' / absent = it applied). */
  promo_issue?: string

  /** Grand total per market, for the muted second line. Same intersection
   *  honesty as everywhere: a market the basket or its promo cannot be
   *  priced in is absent. */
  totals: Money
}

// --- Checkout (E6) ------------------------------------------------------

export interface Address {
  first_name: string
  last_name: string
  phone: string
  street: string
  city: string
  postal_code: string
  country: string
}

export type PaymentMethod = 'card' | 'bank_transfer' | 'cash_on_delivery'
export type PaymentStatus = 'unpaid' | 'paid' | 'refunded'

/**
 * Everything the client CONTRIBUTES to an order — note there is no money in
 * it. Items come from the cart, prices from the server's own tables, the
 * currency from the request; a body that smuggles a total is refused with a
 * 400 before any handler code runs.
 */
export interface CheckoutInput {
  address: Address
  payment_method: PaymentMethod
  delivery_note: string
  leave_with_neighbour: boolean
}

export type OrderStatus = 'pending' | 'confirmed' | 'shipped' | 'delivered' | 'cancelled'

export interface OrderItem {
  name: string
  label: string
  price_minor: number
  qty: number
}

/** A2: one step of the order's recorded history (order_status_events). */
export interface OrderEvent {
  status: OrderStatus
  created_at: string
}

/**
 * A2: one line's fate when a past order is merged back into the cart.
 * `issue` is a CODE the client translates (the promo_issue contract):
 * absent/'' = added in full.
 */
export interface ReorderLine {
  name: string
  label: string
  qty: number
  issue?: 'reduced' | 'out_of_stock' | 'unavailable'
}

export interface ReorderResult {
  lines: ReorderLine[]
}

/** A3: a saved product is a full card plus WHEN the heart was set. */
export interface WishlistEntry extends Product {
  saved_at: string
}

export interface Order {
  id: number
  status: OrderStatus
  created_at: string
  user_email?: string // present in admin responses only
  items: OrderItem[]

  /** The five-figure breakdown (E6). tax_minor is CONTAINED in the subtotal
   *  ("Prices include VAT") — display it, never add it. */
  subtotal_minor: number
  shipping_minor: number
  discount_minor: number
  tax_minor: number
  total_minor: number
  /** E7: the composition of discount_minor — the receipt's two lines. */
  member_discount_minor: number
  promo_discount_minor: number
  /** The frozen text of the redeemed code; absent when none was. */
  promo_code?: string

  payment_method: PaymentMethod
  payment_status: PaymentStatus

  /** The frozen snapshot; absent on orders that predate checkout-with-address. */
  ship_to?: Address
  delivery_note?: string
  leave_with_neighbour: boolean
  /**
   * A2: the recorded timeline, oldest first — what the tracker dates its
   * steps from. Orders older than the events table carry only their
   * backfilled `pending` event; the tracker then shows position without
   * dates rather than inventing them.
   */
  events: OrderEvent[]
  /** A2: any line is a chilled product — the "chilled parcel" tag. */
  has_cold_chain: boolean
  /**
   * What the customer was actually charged in. An order carries ONE currency
   * and no second price — unlike a cart, which is a live thing that can be
   * read in either market. Showing a converted alternative beside a charge
   * invites "but you billed me the other number".
   */
  currency: Currency
  /**
   * The base→order-currency rate at checkout, as an exact decimal STRING
   * (NUMERIC(18,8) server-side; JSON numbers are doubles). Absent for a
   * base-currency order, where no rate applied.
   */
  fx_rate_used?: string
}

// Admin shapes: the public product plus admin-only fields.
export interface AdminProduct extends Product {
  is_active: boolean
}

export interface NewVariantInput {
  sku: string
  label: string
  /**
   * One price per market. The base currency (USD) is required; any other is
   * optional and falls back to a converted price. Replaced the scalar
   * price_minor in E5.
   */
  prices: Money
  stock_qty: number
}

export interface NewProduct {
  category_id: number
  slug: string
  name: string
  description: string
  image_url: string
  variants: NewVariantInput[]
  translations?: Record<string, ProductText>
}

export interface UpdateProduct {
  category_id: number
  name: string
  description: string
  image_url: string
  is_active: boolean
  translations?: Record<string, ProductText>

  // E3 metadata. Optional on the wire, so the existing calls that only
  // toggle is_active keep working unchanged.
  disclaimer?: string
  storage_note?: string
  harvest_note?: string
  shipping_note?: string
  lab_batch?: string
  is_cold_chain?: boolean
}

// --- E3 admin writes ----------------------------------------------------
// Each replaces a whole collection with what the form is showing, which is
// why they are PUTs and why there is no per-row id to patch.

export interface ImageInput {
  id: number
  is_primary: boolean
  /** Alt per locale, including "en" — an image has no parent field for it. */
  alt: Record<string, string>
}

export interface EditorialInput {
  highlights: ProductHighlight[]
  usage_cards: ProductUsageCard[]
}

// Shape of the backend's error envelope (docs/ARCHITECTURE.md).
export interface ApiErrorBody {
  error: {
    code: string
    message: string
    fields?: Record<string, string>
  }
}
