export type MovementType = 'in' | 'out'
export type ProductFilter = 'all' | 'active' | 'inactive' | 'low_stock'
export type OrderStatus = 'CREATED' | 'CANCELLED'

export interface Product {
  id: number
  sku: string
  name: string
  price: number
  stock: number
  is_active: boolean
  created_at: string
  last_movement_at: string | null
}

export interface Movement {
  id: number
  product_id: number
  type: MovementType
  qty: number
  note: string | null
  order_id: number | null
  created_at: string
}

export interface Paged<T> {
  data: T[]
  page: number
  limit: number
  total: number
}

export interface ListResponse<T> {
  data: T[]
}

export interface ProductQuery {
  search: string
  filter: ProductFilter
  page: number
  limit: number
}

export interface CreateProductInput {
  sku: string
  name: string
  price: number
  stock?: number
  is_active?: boolean
}

export interface UpdateProductInput {
  name: string
  price: number
  is_active: boolean
}

export interface StockAdjustInput {
  type: MovementType
  qty: number
  note?: string
}

export interface StockAdjustResult {
  product: Product
  movement: Movement
}

export interface OrderItem {
  id: number
  sku: string
  name: string
  qty: number
  unit_price: number
  line_total: number
}

export interface Order {
  id: number
  customer: string
  status: OrderStatus
  total: number
  created_at: string
  cancelled_at: string | null
  items: OrderItem[]
}

export interface OrderSummary extends Omit<Order, 'items'> {
  item_count: number
}

export interface CreateOrderInput {
  customer: string
  items: {
    sku: string
    qty: number
  }[]
}

export interface Summary {
  total_products: number
  active_products: number
  total_stock_value: number
  low_stock_count: number
}

export interface ErrorDetail {
  field: string
  message: string
}

export interface ApiErrorBody {
  error: {
    code: string
    message: string
    details?: ErrorDetail[]
    sku?: string
    available?: number
  }
}