import { useState } from 'react'
import type { FormEvent } from 'react'

import { ApiError } from '../api/client'
import type { CreateProductInput } from '../api/types'
import { useCreateProduct, useUpdateProduct } from '../api/queries'
import { describeError } from '../lib/errors'
import { normalizeSku, validateProductForm } from '../lib/validation'
import type {
  ProductForm,
  ProductFormErrors,
} from '../lib/validation'
import { Modal } from './Modal'

type ProductFormModalProps =
  | { mode: 'create'; onClose: () => void }
  | { mode: 'edit'; product: import('../api/types').Product; onClose: () => void }

function initialForm(props: ProductFormModalProps): ProductForm {
  if (props.mode === 'edit') {
    return {
      sku: props.product.sku,
      name: props.product.name,
      price: String(props.product.price),
      stock: String(props.product.stock),
      isActive: props.product.is_active,
    }
  }

  return {
    sku: '',
    name: '',
    price: '',
    stock: '',
    isActive: true,
  }
}

export function ProductFormModal(props: ProductFormModalProps) {
  const [form, setForm] = useState<ProductForm>(() => initialForm(props))
  const [errors, setErrors] = useState<ProductFormErrors>({})
  const [generalError, setGeneralError] = useState<string | null>(null)
  const createProduct = useCreateProduct()
  const updateProduct = useUpdateProduct()
  const isPending = createProduct.isPending || updateProduct.isPending
  const isEdit = props.mode === 'edit'

  function updateField<Field extends keyof ProductForm>(
    field: Field,
    value: ProductForm[Field],
  ) {
    setForm((current) => ({ ...current, [field]: value }))

    if (field !== 'isActive') {
      setErrors((current) => ({ ...current, [field]: undefined }))
    }
    setGeneralError(null)
  }

  function handleMutationError(error: unknown) {
    const nextErrors: ProductFormErrors = {}
    const generalErrors: string[] = []

    if (error instanceof ApiError && error.code === 'SKU_ALREADY_EXISTS') {
      nextErrors.sku = error.message
    } else if (
      error instanceof ApiError
      && error.code === 'VALIDATION_FAILED'
      && error.details?.length
    ) {
      for (const detail of error.details) {
        switch (detail.field) {
          case 'sku':
            nextErrors.sku = detail.message
            break
          case 'name':
            nextErrors.name = detail.message
            break
          case 'price':
            nextErrors.price = detail.message
            break
          case 'stock':
            nextErrors.stock = detail.message
            break
          default:
            generalErrors.push(detail.message)
        }
      }
    } else {
      generalErrors.push(describeError(error))
    }

    setErrors(nextErrors)
    setGeneralError(generalErrors.length > 0 ? generalErrors.join('\n') : null)
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setGeneralError(null)

    const nextErrors = validateProductForm(form, props.mode)
    setErrors(nextErrors)
    if (Object.values(nextErrors).some(Boolean)) {
      return
    }

    const name = form.name.trim()
    const price = Number(form.price.trim())

    if (props.mode === 'create') {
      const input: CreateProductInput = {
        sku: normalizeSku(form.sku),
        name,
        price,
        is_active: form.isActive,
      }
      if (form.stock.trim()) {
        input.stock = Number(form.stock.trim())
      }

      createProduct.mutate(input, {
        onSuccess: props.onClose,
        onError: handleMutationError,
      })
      return
    }

    updateProduct.mutate(
      {
        id: props.product.id,
        input: {
          name,
          price,
          is_active: form.isActive,
        },
      },
      {
        onSuccess: props.onClose,
        onError: handleMutationError,
      },
    )
  }

  return (
    <Modal
      title={isEdit ? `Edit product — ${props.product.sku}` : 'New product'}
      onClose={props.onClose}
    >
      <form onSubmit={handleSubmit} noValidate>
        <div>
          <label htmlFor="product-sku">SKU</label>
          <input
            id="product-sku"
            value={form.sku}
            readOnly={isEdit}
            aria-invalid={Boolean(errors.sku)}
            aria-describedby={errors.sku ? 'product-sku-error' : undefined}
            onChange={(event) => updateField('sku', event.target.value)}
          />
          {errors.sku && (
            <p id="product-sku-error" className="row-error">
              {errors.sku}
            </p>
          )}
        </div>

        <div>
          <label htmlFor="product-name">Name</label>
          <input
            id="product-name"
            value={form.name}
            aria-invalid={Boolean(errors.name)}
            aria-describedby={errors.name ? 'product-name-error' : undefined}
            onChange={(event) => updateField('name', event.target.value)}
          />
          {errors.name && (
            <p id="product-name-error" className="row-error">
              {errors.name}
            </p>
          )}
        </div>

        <div>
          <label htmlFor="product-price">Price</label>
          <input
            id="product-price"
            inputMode="numeric"
            value={form.price}
            aria-invalid={Boolean(errors.price)}
            aria-describedby={errors.price ? 'product-price-error' : undefined}
            onChange={(event) => updateField('price', event.target.value)}
          />
          {errors.price && (
            <p id="product-price-error" className="row-error">
              {errors.price}
            </p>
          )}
        </div>

        {!isEdit && (
          <div>
            <label htmlFor="product-stock">Stock</label>
            <input
              id="product-stock"
              inputMode="numeric"
              value={form.stock}
              aria-invalid={Boolean(errors.stock)}
              aria-describedby={errors.stock ? 'product-stock-error' : undefined}
              onChange={(event) => updateField('stock', event.target.value)}
            />
            {errors.stock && (
              <p id="product-stock-error" className="row-error">
                {errors.stock}
              </p>
            )}
          </div>
        )}

        <label>
          <input
            type="checkbox"
            checked={form.isActive}
            onChange={(event) => updateField('isActive', event.target.checked)}
          />
          Active
        </label>

        {generalError && <p role="alert">{generalError}</p>}

        <button type="submit" disabled={isPending}>
          {isEdit ? 'Save changes' : 'Create product'}
        </button>
      </form>
    </Modal>
  )
}
