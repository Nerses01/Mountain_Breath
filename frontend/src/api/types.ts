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

export interface Product {
  id: number
  category_id: number
  slug: string
  name: string
  description: string
  image_url: string
  created_at: string
  variants: ProductVariant[]
}

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

export interface NewCategory {
  slug: string
  name: string
  sort_order: number
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
}

export interface UpdateProduct {
  category_id: number
  name: string
  description: string
  image_url: string
  is_active: boolean
}

// Shape of the backend's error envelope (docs/ARCHITECTURE.md).
export interface ApiErrorBody {
  error: {
    code: string
    message: string
    fields?: Record<string, string>
  }
}
