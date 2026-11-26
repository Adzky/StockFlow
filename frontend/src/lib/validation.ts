import type {
  CreateOrderInput,
  MovementType,
} from '../api/types'

export const QTY_MESSAGE =
  'Quantity must be a whole number from 1 to 1,000,000'

export function normalizeSku(s: string): string {
  return s.trim().toUpperCase()
}

export interface ProductForm {
  sku: string
  name: string
  price: string
  stock: string
  isActive: boolean
}

export type ProductFormErrors = Partial<
  Record<'sku' | 'name' | 'price' | 'stock', string>
>

export function validateProductForm(
  form: ProductForm,
  mode: 'create' | 'edit',
): ProductFormErrors {
  const errors: ProductFormErrors = {}

  if (mode === 'create') {
    const sku = normalizeSku(form.sku)

    if (!sku) {
      errors.sku = 'is required'
    } else if (sku.length > 64) {
      errors.sku = 'must be at most 64 characters'
    } else if (!/^[A-Z0-9-]+$/.test(sku)) {
      errors.sku = 'may contain only A-Z, 0-9 and -'
    }
  }

  const name = form.name.trim()
  if (!name) {
    errors.name = 'is required'
  } else if (Array.from(name).length > 200) {
    errors.name = 'must be at most 200 characters'
  }

  const price = form.price.trim()
  const priceNumber = Number(price)
  if (!price) {
    errors.price = 'is required'
  } else if (
    !/^-?\d+$/.test(price)
    || priceNumber < 0
    || priceNumber > 1_000_000_000
  ) {
    errors.price = 'must be between 0 and 1000000000'
  }

  if (mode === 'create' && form.stock.trim()) {
    const stock = form.stock.trim()
    const stockNumber = Number(stock)
    if (!/^\d+$/.test(stock) || stockNumber > 1_000_000) {
      errors.stock = 'must be between 0 and 1000000'
    }
  }

  return errors
}

export function parseQty(raw: string): number | null {
  const value = raw.trim()

  if (!/^\d+$/.test(value)) {
    return null
  }

  const qty = Number(value)
  return qty >= 1 && qty <= 1_000_000 ? qty : null
}

export interface StockAdjustForm {
  type: MovementType
  qty: string
  note: string
}

export interface StockAdjustErrors {
  qty?: string
  note?: string
}

export function validateStockAdjust(
  form: StockAdjustForm,
): StockAdjustErrors {
  const errors: StockAdjustErrors = {}

  if (parseQty(form.qty) === null) {
    errors.qty = QTY_MESSAGE
  }

  if (Array.from(form.note.trim()).length > 500) {
    errors.note = 'Note must be at most 500 characters'
  }

  return errors
}

export interface OrderItemDraft {
  sku: string
  qty: string
}

export interface OrderDraft {
  customer: string
  items: OrderItemDraft[]
}

export interface OrderFormErrors {
  customer?: string
  items: Record<number, string>
  general: string[]
}

export function validateOrderDraft(draft: OrderDraft): OrderFormErrors {
  const errors: OrderFormErrors = {
    items: {},
    general: [],
  }

  if (draft.customer.trim() === '') {
    errors.customer = 'Customer name is required'
  }

  if (draft.items.length === 0) {
    errors.general.push('Add at least one item')
  }

  const firstRowBySku = new Map<string, number>()

  draft.items.forEach((item, index) => {
    const sku = normalizeSku(item.sku)

    if (sku === '') {
      errors.items[index] = 'SKU is required'
      return
    }

    const firstRow = firstRowBySku.get(sku)
    if (firstRow !== undefined) {
      errors.items[index] = `Duplicate SKU — already in row ${firstRow + 1}`
      return
    }

    firstRowBySku.set(sku, index)

    if (parseQty(item.qty) === null) {
      errors.items[index] = QTY_MESSAGE
    }
  })

  return errors
}

export function hasErrors(
  errors: StockAdjustErrors | OrderFormErrors,
): boolean {
  if ('general' in errors) {
    return Boolean(errors.customer)
      || Object.keys(errors.items).length > 0
      || errors.general.length > 0
  }

  return Object.values(errors).some(Boolean)
}

export function toCreateOrderInput(
  draft: OrderDraft,
): CreateOrderInput {
  return {
    customer: draft.customer.trim(),
    items: draft.items.map((item) => {
      const qty = parseQty(item.qty)

      if (qty === null) {
        throw new Error(QTY_MESSAGE)
      }

      return {
        sku: normalizeSku(item.sku),
        qty,
      }
    }),
  }
}