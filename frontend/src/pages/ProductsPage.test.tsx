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
})