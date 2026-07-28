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

// Shape of the backend's error envelope (docs/ARCHITECTURE.md).
export interface ApiErrorBody {
  error: {
    code: string
    message: string
    fields?: Record<string, string>
  }
}
