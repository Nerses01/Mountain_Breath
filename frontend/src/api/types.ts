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
}

export interface Credentials {
  email: string
  password: string
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
  total_minor: number
  /**
   * The basket summed independently in each market — never converted from
   * one to another, because rounding a sum is not the sum of roundings.
   * A market any line cannot be priced in is absent rather than understated.
   */
  totals: Money
}

export type OrderStatus = 'pending' | 'confirmed' | 'shipped' | 'delivered' | 'cancelled'

export interface OrderItem {
  name: string
  label: string
  price_minor: number
  qty: number
}

export interface Order {
  id: number
  status: OrderStatus
  total_minor: number
  created_at: string
  user_email?: string // present in admin responses only
  items: OrderItem[]
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
