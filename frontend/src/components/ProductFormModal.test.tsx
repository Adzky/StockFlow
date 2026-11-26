import {
  cleanup,
  render,
  screen,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query'
import type { PropsWithChildren, ReactElement } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api, ApiError } from '../api/client'
import type { Product } from '../api/types'
import { ProductFormModal } from './ProductFormModal'

const product: Product = {
  id: 1,
  sku: 'BR-001',
  name: 'Beras',
  price: 78_000,
  stock: 15,
  is_active: true,
  created_at: '2026-11-25T12:00:00Z',
  last_movement_at: '2026-11-25T12:00:00Z',
}

const queryClients: QueryClient[] = []

function renderModal(element: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  queryClients.push(queryClient)

  function Wrapper({ children }: PropsWithChildren) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    )
  }

  return render(element, { wrapper: Wrapper })
}

async function fillCreateForm(
  user: ReturnType<typeof userEvent.setup>,
  sku = 'BRS-2',
) {
  await user.type(screen.getByLabelText('SKU'), sku)
  await user.type(screen.getByLabelText('Name'), 'Beras Baru')
  await user.type(screen.getByLabelText('Price'), '90000')
  await user.type(screen.getByLabelText('Stock'), '24')
}

afterEach(() => {
  cleanup()
  queryClients.forEach((queryClient) => queryClient.clear())
  queryClients.length = 0
  vi.restoreAllMocks()
})

describe('ProductFormModal', () => {
  it('shows a duplicate-SKU conflict under the SKU field', async () => {
    vi.spyOn(api, 'createProduct').mockRejectedValue(
      new ApiError(
        409,
        'SKU_ALREADY_EXISTS',
        'sku "BRS-2" already exists',
      ),
    )
    const user = userEvent.setup()

    renderModal(
      <ProductFormModal mode="create" onClose={vi.fn()} />,
    )
    await fillCreateForm(user)
    await user.click(screen.getByRole('button', { name: 'Create product' }))

    expect(
      await screen.findByText('sku "BRS-2" already exists'),
    ).toBeInTheDocument()
    expect(screen.getByLabelText('SKU')).toHaveAttribute(
      'aria-invalid',
      'true',
    )
  })

  it('maps a 400 field validation error to that form field', async () => {
    vi.spyOn(api, 'createProduct').mockRejectedValue(
      new ApiError(400, 'VALIDATION_FAILED', 'validation failed', {
        details: [{ field: 'name', message: 'is required' }],
      }),
    )
    const user = userEvent.setup()

    renderModal(
      <ProductFormModal mode="create" onClose={vi.fn()} />,
    )
    await fillCreateForm(user)
    await user.click(screen.getByRole('button', { name: 'Create product' }))

    expect(await screen.findByText('is required')).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toHaveAttribute(
      'aria-invalid',
      'true',
    )
  })

  it('creates a normalized product and closes on success', async () => {
    const createProduct = vi
      .spyOn(api, 'createProduct')
      .mockResolvedValue(product)
    const onClose = vi.fn()
    const user = userEvent.setup()

    renderModal(
      <ProductFormModal mode="create" onClose={onClose} />,
    )
    await fillCreateForm(user, ' brs-2 ')
    await user.click(screen.getByRole('button', { name: 'Create product' }))

    expect(createProduct).toHaveBeenCalledWith({
      sku: 'BRS-2',
      name: 'Beras Baru',
      price: 90000,
      stock: 24,
      is_active: true,
    })
    expect(onClose).toHaveBeenCalledOnce()
  })

  it('keeps SKU read-only and hides stock when editing', async () => {
    const updateProduct = vi
      .spyOn(api, 'updateProduct')
      .mockResolvedValue(product)
    const onClose = vi.fn()
    const user = userEvent.setup()

    renderModal(
      <ProductFormModal
        mode="edit"
        product={product}
        onClose={onClose}
      />,
    )

    expect(screen.getByLabelText('SKU')).toHaveValue('BR-001')
    expect(screen.getByLabelText('SKU')).toHaveAttribute('readonly')
    expect(screen.queryByLabelText('Stock')).not.toBeInTheDocument()

    await user.clear(screen.getByLabelText('Name'))
    await user.type(screen.getByLabelText('Name'), 'Beras Pilihan')
    await user.clear(screen.getByLabelText('Price'))
    await user.type(screen.getByLabelText('Price'), '80000')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(updateProduct).toHaveBeenCalledWith(1, {
      name: 'Beras Pilihan',
      price: 80000,
      is_active: true,
    })
    expect(onClose).toHaveBeenCalledOnce()
  })
})
