import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from '@testing-library/react'
import {
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query'
import type { PropsWithChildren } from 'react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api, ApiError } from '../api/client'
import type { Paged, Product, Summary } from '../api/types'
import { ProductsPage } from './ProductsPage'

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

const summary: Summary = {
  total_products: 15,
  active_products: 13,
  total_stock_value: 15_069_000,
  low_stock_count: 4,
}

function productResponse(data: Product[]): Paged<Product> {
  return {
    data,
    page: 1,
    limit: 20,
    total: data.length,
  }
}

const queryClients: QueryClient[] = []

function renderProductsPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })
  queryClients.push(queryClient)

  vi.spyOn(api, 'getSummary').mockResolvedValue(summary)

  function Wrapper({ children }: PropsWithChildren) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    )
  }

  render(<ProductsPage />, { wrapper: Wrapper })
}

afterEach(() => {
  cleanup()
  queryClients.forEach((queryClient) => queryClient.clear())
  queryClients.length = 0
  vi.restoreAllMocks()
  vi.useRealTimers()
})

describe('ProductsPage', () => {
  it('opens the create form from the New product button', async () => {
    vi.spyOn(api, 'listProducts').mockResolvedValue(
      productResponse([product]),
    )

    renderProductsPage()

    await screen.findByRole('row', { name: /BR-001/ })
    fireEvent.click(screen.getByRole('button', { name: 'New product' }))

    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByLabelText('SKU')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Create product' }),
    ).toBeInTheDocument()
  })

  it('renders a product row with formatted price and active status', async () => {
    vi.spyOn(api, 'listProducts').mockResolvedValue(
      productResponse([product]),
    )

    renderProductsPage()

    const row = await screen.findByRole('row', { name: /BR-001/ })
    expect(within(row).getByText('Rp 78.000')).toBeInTheDocument()
    expect(within(row).getByText('Active')).toBeInTheDocument()
  })

  it('shows the empty state when no products match', async () => {
    vi.spyOn(api, 'listProducts').mockResolvedValue(
      productResponse([]),
    )

    renderProductsPage()

    expect(await screen.findByText('No products match')).toBeInTheDocument()
  })

  it('shows Retry after an error and renders rows after retry succeeds', async () => {
    const listProducts = vi
      .spyOn(api, 'listProducts')
      .mockRejectedValueOnce(
        new ApiError(500, 'UNKNOWN', 'Server error'),
      )
      .mockResolvedValueOnce(productResponse([product]))

    renderProductsPage()

    expect(await screen.findByRole('alert')).toHaveTextContent('Server error')

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    expect(
      await screen.findByRole('row', { name: /BR-001/ }),
    ).toBeInTheDocument()
    expect(listProducts).toHaveBeenCalledTimes(2)
  })

  it('debounces product search by 300 ms and requests page 1', async () => {
    vi.useFakeTimers()

    const listProducts = vi
      .spyOn(api, 'listProducts')
      .mockResolvedValue(productResponse([product]))

    renderProductsPage()

    fireEvent.change(screen.getByLabelText(/search/i), {
      target: { value: 'beras' },
    })

    await act(async () => {
      await vi.advanceTimersByTimeAsync(299)
    })

    expect(listProducts).not.toHaveBeenCalledWith(
      expect.objectContaining({ search: 'beras' }),
    )

    await act(async () => {
      await vi.advanceTimersByTimeAsync(1)
    })

    expect(listProducts).toHaveBeenCalledWith({
      search: 'beras',
      filter: 'all',
      page: 1,
      limit: 20,
    })
  })

  it('opens the selected product in edit mode', async () => {
    vi.spyOn(api, 'listProducts').mockResolvedValue(
      productResponse([product]),
    )

    renderProductsPage()

    const row = await screen.findByRole('row', { name: /BR-001/ })
    fireEvent.click(within(row).getByRole('button', { name: 'Edit' }))

    expect(screen.getByLabelText('SKU')).toHaveValue('BR-001')
    expect(screen.getByLabelText('SKU')).toHaveAttribute('readonly')
    expect(screen.queryByLabelText('Stock')).not.toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Save changes' }),
    ).toBeInTheDocument()
  })

  it('shows the history message when product deletion returns 409', async () => {
    vi.spyOn(api, 'listProducts').mockResolvedValue(
      productResponse([product]),
    )
    const deleteProduct = vi.spyOn(api, 'deleteProduct').mockRejectedValue(
      new ApiError(
        409,
        'PRODUCT_HAS_HISTORY',
        'product has stock history and cannot be deleted',
      ),
    )

    renderProductsPage()

    const row = await screen.findByRole('row', { name: /BR-001/ })
    fireEvent.click(within(row).getByRole('button', { name: 'Delete' }))
    expect(screen.getByRole('dialog')).toHaveTextContent('Delete BR-001?')
    fireEvent.click(
      screen.getByRole('button', { name: 'Confirm delete' }),
    )

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "This product has stock history and can't be deleted. Deactivate it instead.",
    )
    expect(deleteProduct).toHaveBeenCalledWith(product.id)
  })

  it('removes a product from the list after deletion succeeds', async () => {
    vi.spyOn(api, 'listProducts')
      .mockResolvedValueOnce(productResponse([product]))
      .mockResolvedValue(productResponse([]))
    const deleteProduct = vi
      .spyOn(api, 'deleteProduct')
      .mockResolvedValue(undefined)

    renderProductsPage()

    const row = await screen.findByRole('row', { name: /BR-001/ })
    fireEvent.click(within(row).getByRole('button', { name: 'Delete' }))
    fireEvent.click(
      screen.getByRole('button', { name: 'Confirm delete' }),
    )

    expect(await screen.findByText('No products match')).toBeInTheDocument()
    expect(deleteProduct).toHaveBeenCalledWith(product.id)
  })
})