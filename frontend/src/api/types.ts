// Wire types: these mirror the Go DTOs in backend/internal/api (snake_case
// fields, exactly as JSON arrives). If the backend contract changes, change
// it here too — the compiler then points at every affected component.

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
  price_minor: number
  stock_qty: number
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
export type ProductSort = 'popular' | 'price_asc' | 'price_desc' | 'newest'

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
  price_minor: number
  stock_qty: number
  qty: number
  line_total_minor: number
}

export interface Cart {
  items: CartItem[]
  total_minor: number
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
}

// Admin shapes: the public product plus admin-only fields.
export interface AdminProduct extends Product {
  is_active: boolean
}

export interface NewVariantInput {
  sku: string
  label: string
  price_minor: number
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
}

// Shape of the backend's error envelope (docs/ARCHITECTURE.md).
export interface ApiErrorBody {
  error: {
    code: string
    message: string
    fields?: Record<string, string>
  }
}
