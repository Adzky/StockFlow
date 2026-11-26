import type {
  ApiErrorBody,
  CreateOrderInput,
  CreateProductInput,
  ListResponse,
  Movement,
  Order,
  OrderSummary,
  Paged,
  Product,
  ProductQuery,
  StockAdjustInput,
  StockAdjustResult,
  Summary,
  UpdateProductInput,
  ErrorDetail,
} from './types'

type Query = Record<string, string | number | undefined>

function buildUrl(path: string, query?: Query): string {
  const params = new URLSearchParams()

  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== '') {
      params.set(key, String(value))
    }
  }

  const queryString = params.toString()
  return queryString ? `${path}?${queryString}` : path
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string
  readonly details?: ErrorDetail[]
  readonly sku?: string
  readonly available?: number

  constructor(
    status: number,
    code: string,
    message: string,
    extra?: {
      details?: ErrorDetail[]
      sku?: string
      available?: number
    },
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.details = extra?.details
    this.sku = extra?.sku
    this.available = extra?.available
  }
}

function isErrorBody(value: unknown): value is ApiErrorBody {
  if (typeof value !== 'object' || value === null) {
    return false
  }

  if (!('error' in value)) {
    return false
  }

  const error: unknown = value.error
  return (
    typeof error === 'object'
    && error !== null
    && 'code' in error
    && typeof error.code === 'string'
  )
}

async function request<T>(
  method: string,
  path: string,
  options: {
    query?: Query
    body?: unknown
  } = {},
): Promise<T> {
  let response: Response

  try {
    response = await fetch(buildUrl(path, options.query), {
      method,
      headers:
        options.body === undefined
          ? undefined
          : { 'Content-Type': 'application/json' },
      body:
        options.body === undefined
          ? undefined
          : JSON.stringify(options.body),
    })
  } catch {
    throw new ApiError(
      0,
      'NETWORK_ERROR',
      'Cannot reach the server. Is the backend running?',
    )
  }

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  let parsed: unknown = null

  if (text) {
    try {
      parsed = JSON.parse(text)
    } catch {
      parsed = null
    }
  }

  if (!response.ok) {
    if (isErrorBody(parsed)) {
      const { code, message, details, sku, available } = parsed.error
      throw new ApiError(
        response.status,
        code,
        message,
        { details, sku, available },
      )
    }

    throw new ApiError(
      response.status,
      'UNKNOWN',
      `Server error (HTTP ${response.status})`,
    )
  }

  return parsed as T
}

export const api = {
  listProducts: (query: ProductQuery): Promise<Paged<Product>> =>
    request('GET', '/api/products', {
      query: {
        search: query.search,
        filter: query.filter,
        page: query.page,
        limit: query.limit,
      },
    }),

  createProduct: (input: CreateProductInput): Promise<Product> =>
    request('POST', '/api/products', { body: input }),

  updateProduct: (
    id: number,
    input: UpdateProductInput,
  ): Promise<Product> =>
    request('PUT', `/api/products/${id}`, { body: input }),

  deleteProduct: (id: number): Promise<void> =>
    request('DELETE', `/api/products/${id}`),

  adjustStock: (
    id: number,
    input: StockAdjustInput,
  ): Promise<StockAdjustResult> =>
    request(
      'POST',
      `/api/products/${id}/stock`,
      { body: input },
    ),

  listMovements: (id: number): Promise<ListResponse<Movement>> =>
    request('GET', `/api/products/${id}/movements`),

  createOrder: (input: CreateOrderInput): Promise<Order> =>
    request('POST', '/api/orders', { body: input }),

  getOrder: (id: number): Promise<Order> =>
    request('GET', `/api/orders/${id}`),

  listOrders: (
    page: number,
    limit: number,
  ): Promise<Paged<OrderSummary>> =>
    request('GET', '/api/orders', {
      query: { page, limit },
    }),

  cancelOrder: (id: number): Promise<Order> =>
    request('POST', `/api/orders/${id}/cancel`),

  getSummary: (): Promise<Summary> =>
    request('GET', '/api/summary'),
}