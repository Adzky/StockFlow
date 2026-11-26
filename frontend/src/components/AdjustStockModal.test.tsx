import type { ReactNode } from 'react'
import {
  cleanup,
  render,
  screen,
  waitFor,
} from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import {
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api, ApiError } from '../api/client'
import type { Product, StockAdjustResult } from '../api/types'
import { AdjustStockModal } from './AdjustStockModal'

const product: Product = {
  id: 7,
  sku: 'TLR-AYM-30',
  name: 'Telur ayam',
  price: 2500,
  stock: 6,
  is_active: true,
  created_at: '2026-11-25T12:00:00Z',
  last_movement_at: '2026-11-25T12:00:00Z',
}

const result: StockAdjustResult = {
  product: { ...product, stock: 9 },
  movement: {
    id: 12,
    product_id: 7,
    type: 'in',
    qty: 3,
    note: 'restock',
    order_id: null,
    created_at: '2026-11-26T12:00:00Z',
  },
}

const queryClients: QueryClient[] = []

function renderWithClient(ui: ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  queryClients.push(queryClient)

  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>,
  )
}

afterEach(() => {
  cleanup()
  queryClients.forEach((queryClient) => queryClient.clear())
  queryClients.length = 0
  vi.restoreAllMocks()
})

describe('AdjustStockModal', () => {
  it('shows available stock after a 409 and stays open', async () => {
    vi.spyOn(api, 'adjustStock').mockRejectedValue(
      new ApiError(409, 'INSUFFICIENT_STOCK', 'insufficient stock', {
        available: 3,
      }),
    )

    const user = userEvent.setup()
    const onClose = vi.fn()

    renderWithClient(
      <AdjustStockModal
        product={product}
        onClose={onClose}
        onSaved={vi.fn()}
      />,
    )

    await user.click(screen.getByLabelText('Out'))
    await user.type(screen.getByLabelText('Quantity'), '5')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Insufficient stock — only 3 available',
    )
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('disables Save while submitting so a second click does not submit again', async () => {
    const adjustStock = vi.spyOn(api, 'adjustStock').mockImplementation(
      () => new Promise<StockAdjustResult>(() => {}),
    )

    const user = userEvent.setup()

    renderWithClient(
      <AdjustStockModal
        product={product}
        onClose={vi.fn()}
        onSaved={vi.fn()}
      />,
    )

    await user.type(screen.getByLabelText('Quantity'), '3')

    const saveButton = screen.getByRole('button', { name: 'Save' })
    await user.click(saveButton)

    await waitFor(() => expect(saveButton).toBeDisabled())
    await user.click(saveButton)

    expect(adjustStock).toHaveBeenCalledTimes(1)
  })

  it('shows the quantity validation error and disables Save for decimals', async () => {
    const user = userEvent.setup()

    renderWithClient(
      <AdjustStockModal
        product={product}
        onClose={vi.fn()}
        onSaved={vi.fn()}
      />,
    )

    await user.type(screen.getByLabelText('Quantity'), '1.5')
    await user.click(screen.getByLabelText('Note'))

    expect(
      screen.getByText('Quantity must be a whole number from 1 to 1,000,000'),
    ).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })

  it('calls onSaved before onClose when the adjustment succeeds', async () => {
    vi.spyOn(api, 'adjustStock').mockResolvedValue(result)

    const calls: string[] = []
    const onSaved = vi.fn((_savedResult: StockAdjustResult) => {
      calls.push('saved')
    })
    const onClose = vi.fn(() => {
      calls.push('closed')
    })
    const user = userEvent.setup()

    renderWithClient(
      <AdjustStockModal
        product={product}
        onClose={onClose}
        onSaved={onSaved}
      />,
    )

    await user.type(screen.getByLabelText('Quantity'), '3')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))

    expect(onSaved).toHaveBeenCalledWith(result)
    expect(calls).toEqual(['saved', 'closed'])
  })

  it('shows other API errors in an alert and stays open', async () => {
    vi.spyOn(api, 'adjustStock').mockRejectedValue(
      new ApiError(500, 'UNKNOWN', 'Server error'),
    )

    const onClose = vi.fn()
    const user = userEvent.setup()

    renderWithClient(
      <AdjustStockModal
        product={product}
        onClose={onClose}
        onSaved={vi.fn()}
      />,
    )

    await user.type(screen.getByLabelText('Quantity'), '3')
    await user.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Server error')
    expect(onClose).not.toHaveBeenCalled()
  })
})