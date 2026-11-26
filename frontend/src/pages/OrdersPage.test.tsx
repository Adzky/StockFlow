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
import { afterEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter, Route, Routes } from 'react-router-dom'

import { api } from '../api/client'
import type { OrderSummary, Paged } from '../api/types'
import { OrdersPage } from './OrdersPage'

const order: OrderSummary = {
  id: 1,
  customer: 'Toko',
  status: 'CREATED',
  total: 8000,
  created_at: '2026-11-26T10:00:00Z',
  cancelled_at: null,
  item_count: 2,
}

function ordersResponse(
  data: OrderSummary[],
  total = data.length,
  page = 1,
): Paged<OrderSummary> {
  return { data, page, limit: 20, total }
}

const queryClients: QueryClient[] = []

function renderOrdersPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  })
  queryClients.push(queryClient)

  return render(
    <MemoryRouter initialEntries={['/orders']}>
      <QueryClientProvider client={queryClient}>
        <Routes>
          <Route path="/orders" element={<OrdersPage />} />
          <Route path="/orders/:id" element={<p>Order detail</p>} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  )
}

afterEach(() => {
  cleanup()
  queryClients.forEach((queryClient) => queryClient.clear())
  queryClients.length = 0
  vi.restoreAllMocks()
})

describe('OrdersPage', () => {
  it('renders orders and requests the next page', async () => {
    const listOrders = vi
      .spyOn(api, 'listOrders')
      .mockImplementation(async (page) => ordersResponse([order], 21, page))
    const user = userEvent.setup()

    renderOrdersPage()

    expect(await screen.findByRole('link', { name: '#1' })).toBeInTheDocument()
    expect(screen.getByText('Rp 8.000')).toBeInTheDocument()
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Next' }))

    expect(await screen.findByText('Page 2 of 2')).toBeInTheDocument()
    expect(listOrders).toHaveBeenLastCalledWith(2, 20)
  })

  it('shows an empty state when there are no orders', async () => {
    vi.spyOn(api, 'listOrders').mockResolvedValue(ordersResponse([]))

    renderOrdersPage()

    expect(await screen.findByText('No orders yet')).toBeInTheDocument()
  })

  it('shows an error with Retry and recovers when the list request succeeds', async () => {
    vi.spyOn(api, 'listOrders')
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce(ordersResponse([order]))
    const user = userEvent.setup()

    renderOrdersPage()

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Something went wrong',
    )
    await user.click(screen.getByRole('button', { name: 'Retry' }))

    expect(await screen.findByRole('link', { name: '#1' })).toBeInTheDocument()
  })
})
