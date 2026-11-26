import { useState } from 'react'
import type { FormEvent } from 'react'

import { ApiError } from '../api/client'
import { useAdjustStock } from '../api/queries'
import type { Product, StockAdjustResult } from '../api/types'
import {
  hasErrors,
  parseQty,
  validateStockAdjust,
} from '../lib/validation'
import type { StockAdjustForm } from '../lib/validation'
import { describeError, insufficientStockMessage } from '../lib/errors'
import { Modal } from './Modal'

export function AdjustStockModal({
  product,
  onClose,
  onSaved,
}: {
  product: Product
  onClose: () => void
  onSaved: (result: StockAdjustResult) => void
}) {
  const [form, setForm] = useState<StockAdjustForm>({
    type: 'in',
    qty: '',
    note: '',
  })
  const [touched, setTouched] = useState({ qty: false, note: false })
  const [submitted, setSubmitted] = useState(false)
  const adjustment = useAdjustStock()

  const errors = validateStockAdjust(form)
  const invalid = hasErrors(errors)
  const showQtyError = submitted || touched.qty
  const showNoteError = submitted || touched.note

  const mutationError = adjustment.error
  const errorMessage =
    mutationError instanceof ApiError
      && mutationError.status === 409
      && mutationError.code === 'INSUFFICIENT_STOCK'
      ? insufficientStockMessage(mutationError.available ?? 0)
      : mutationError
        ? describeError(mutationError)
        : null

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setSubmitted(true)
    setTouched({ qty: true, note: true })

    const qty = parseQty(form.qty)
    if (qty === null || invalid) {
      return
    }

    adjustment.mutate(
      {
        id: product.id,
        input: {
          type: form.type,
          qty,
          note: form.note.trim() || undefined,
        },
      },
      {
        onSuccess: (result) => {
          onSaved(result)
          onClose()
        },
      },
    )
  }

  return (
    <Modal title={`Adjust stock — ${product.name}`} onClose={onClose}>
      <form onSubmit={handleSubmit} noValidate>
        <p>Current stock: {product.stock}</p>

        <fieldset>
          <legend>Adjustment type</legend>
          <label>
            <input
              type="radio"
              name="stock-adjust-type"
              value="in"
              checked={form.type === 'in'}
              onChange={() => setForm((current) => ({ ...current, type: 'in' }))}
            />
            In
          </label>
          <label>
            <input
              type="radio"
              name="stock-adjust-type"
              value="out"
              checked={form.type === 'out'}
              onChange={() => setForm((current) => ({ ...current, type: 'out' }))}
            />
            Out
          </label>
        </fieldset>

        <div>
          <label htmlFor="stock-adjust-quantity">Quantity</label>
          <input
            id="stock-adjust-quantity"
            type="text"
            inputMode="numeric"
            value={form.qty}
            aria-invalid={showQtyError && Boolean(errors.qty)}
            aria-describedby={showQtyError && errors.qty ? 'quantity-error' : undefined}
            onChange={(event) =>
              setForm((current) => ({ ...current, qty: event.target.value }))
            }
            onBlur={() => setTouched((current) => ({ ...current, qty: true }))}
          />
          {showQtyError && errors.qty && (
            <p id="quantity-error" className="row-error">
              {errors.qty}
            </p>
          )}
        </div>

        <div>
          <label htmlFor="stock-adjust-note">Note</label>
          <textarea
            id="stock-adjust-note"
            value={form.note}
            aria-invalid={showNoteError && Boolean(errors.note)}
            aria-describedby={showNoteError && errors.note ? 'note-error' : undefined}
            onChange={(event) =>
              setForm((current) => ({ ...current, note: event.target.value }))
            }
            onBlur={() => setTouched((current) => ({ ...current, note: true }))}
          />
          {showNoteError && errors.note && (
            <p id="note-error" className="row-error">
              {errors.note}
            </p>
          )}
        </div>

        {errorMessage && (
          <div className="alert" role="alert">
            {errorMessage}
          </div>
        )}

        <button type="submit" disabled={invalid || adjustment.isPending}>
          Save
        </button>
      </form>
    </Modal>
  )
}