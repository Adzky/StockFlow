import { ApiError } from '../api/client'
import type {
  OrderDraft,
  OrderFormErrors,
} from './validation'
import { normalizeSku } from './validation'

export function describeError(err: unknown): string {
  return err instanceof ApiError
    ? err.message
    : 'Something went wrong'
}

export function insufficientStockMessage(available: number): string {
  return `Insufficient stock — only ${available} available`
}

export function mapOrderError(
  err: unknown,
  draft: OrderDraft,
): OrderFormErrors {
  const errors: OrderFormErrors = {
    items: {},
    general: [],
  }

  if (!(err instanceof ApiError)) {
    errors.general.push('Something went wrong')
    return errors
  }

  if (err.code === 'INSUFFICIENT_STOCK') {
    if (err.available === undefined) {
      errors.general.push(err.message)
      return errors
    }

    const rowIndex = draft.items.findIndex(
      (item) => normalizeSku(item.sku) === err.sku,
    )

    if (rowIndex === -1) {
      errors.general.push(insufficientStockMessage(err.available))
    } else {
      errors.items[rowIndex] = `Only ${err.available} available`
    }

    return errors
  }

  if (err.code === 'VALIDATION_FAILED') {
    for (const detail of err.details ?? []) {
      const rowMatch = /^items\[(\d+)\]\.(sku|qty)$/.exec(detail.field)

      if (rowMatch) {
        const rowIndex = Number(rowMatch[1])

        if (errors.items[rowIndex] === undefined) {
          errors.items[rowIndex] = detail.message
        }
      } else if (detail.field === 'customer') {
        errors.customer = detail.message
      } else {
        errors.general.push(detail.message)
      }
    }

    return errors
  }

  errors.general.push(err.message)
  return errors
}