import { useState } from 'react'
import type { FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'

import { useCreateOrder, useProducts } from '../api/queries'
import { mapOrderError } from '../lib/errors'
import type { OrderDraft, OrderFormErrors } from '../lib/validation'
import {
  hasErrors,
  toCreateOrderInput,
  validateOrderDraft,
} from '../lib/validation'
import { ErrorState } from '../components/StateViews'

export function CreateOrderPage() {
  const navigate = useNavigate()
  const createOrder = useCreateOrder()
  const productsQuery = useProducts({
    search: '',
    filter: 'active',
    page: 1,
    limit: 100,
  })

  const [draft, setDraft] = useState<OrderDraft>({
    customer: '',
    items: [{ sku: '', qty: '' }],
  })
  const [errors, setErrors] = useState<OrderFormErrors>({
    items: {},
    general: [],
  })

  function updateItem(index: number, field: 'sku' | 'qty', value: string) {
    setDraft((current) => ({
      ...current,
      items: current.items.map((item, itemIndex) =>
        itemIndex === index ? { ...item, [field]: value } : item,
      ),
    }))

    setErrors((current) => {
      const items = { ...current.items }
      delete items[index]
      return { ...current, items }
    })
  }

  function removeItem(index: number) {
    setDraft((current) => ({
      ...current,
      items: current.items.filter((_item, itemIndex) => itemIndex !== index),
    }))

    setErrors((current) => {
      const items: Record<number, string> = {}
      for (const [indexText, message] of Object.entries(current.items)) {
        const oldIndex = Number(indexText)
        if (oldIndex < index) {
          items[oldIndex] = message
        } else if (oldIndex > index) {
          items[oldIndex - 1] = message
        }
      }
      return { ...current, items }
    })
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()

    const nextErrors = validateOrderDraft(draft)
    setErrors(nextErrors)
    if (hasErrors(nextErrors)) {
      return
    }

    createOrder.mutate(toCreateOrderInput(draft), {
      onSuccess: (order) => {
        navigate(`/orders/${order.id}`, { state: { justCreated: true } })
      },
      onError: (error) => {
        setErrors(mapOrderError(error, draft))
      },
    })
  }

  return (
    <main className="create-order-page">
      <h1>New order</h1>

      {productsQuery.isError && (
        <ErrorState
          error={productsQuery.error}
          onRetry={() => {
            void productsQuery.refetch()
          }}
        />
      )}

      {errors.general.length > 0 && (
        <div role="alert">
          {errors.general.map((message, index) => (
            <p key={`${index}-${message}`}>{message}</p>
          ))}
        </div>
      )}

      <form onSubmit={handleSubmit} noValidate>
        <div>
          <label htmlFor="order-customer">Customer</label>
          <input
            id="order-customer"
            value={draft.customer}
            onChange={(event) => {
              const customer = event.target.value
              setDraft((current) => ({ ...current, customer }))
              setErrors((current) => ({ ...current, customer: undefined }))
            }}
            aria-invalid={Boolean(errors.customer)}
            aria-describedby={errors.customer ? 'customer-error' : undefined}
          />
          {errors.customer && (
            <p id="customer-error" className="row-error">
              {errors.customer}
            </p>
          )}
        </div>

        <datalist id="active-products">
          {productsQuery.data?.data.map((product) => (
            <option key={product.id} value={product.sku}>
              {product.name}
            </option>
          ))}
        </datalist>

        {draft.items.map((item, index) => {
          const rowError = errors.items[index]
          const rowClassName = rowError
            ? 'order-item-row row-error'
            : 'order-item-row'

          return (
            <div className={rowClassName} key={index}>
              <div>
                <label htmlFor={`order-sku-${index}`}>SKU</label>
                <input
                  id={`order-sku-${index}`}
                  list="active-products"
                  value={item.sku}
                  aria-invalid={Boolean(rowError)}
                  aria-describedby={rowError ? `item-error-${index}` : undefined}
                  onChange={(event) =>
                    updateItem(index, 'sku', event.target.value)
                  }
                />
              </div>

              <div>
                <label htmlFor={`order-qty-${index}`}>Qty</label>
                <input
                  id={`order-qty-${index}`}
                  inputMode="numeric"
                  value={item.qty}
                  onChange={(event) =>
                    updateItem(index, 'qty', event.target.value)
                  }
                />
              </div>

              <button
                type="button"
                aria-label={`Remove item ${index + 1}`}
                disabled={draft.items.length === 1}
                onClick={() => removeItem(index)}
              >
                Remove
              </button>

              {rowError && (
                <p id={`item-error-${index}`} role="alert">
                  {rowError}
                </p>
              )}
            </div>
          )
        })}

        <button
          type="button"
          onClick={() =>
            setDraft((current) => ({
              ...current,
              items: [...current.items, { sku: '', qty: '' }],
            }))
          }
        >
          Add item
        </button>

        <button type="submit" disabled={createOrder.isPending}>
          Create order
        </button>
      </form>
    </main>
  )
}
