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
import {
  MemoryRouter,
  Route,
  Routes,
} from 'react-router-dom'

import { api, ApiError } from '../api/client'
import type { Order } from '../api/types'
import { OrderDetailPage } from './OrderDetailPage'

const createdOrder: Order = {
  id: 1,
  customer: 'Toko',
  status: 'CREATED',
  total: 8000,
  created_at: '2026-11-26T10:00:00Z',
  cancelled_at: null,
  items: [
    {
      id: 1,
      sku: 'BRS-1',
      name: 'Beras',
      qty: 2,
      unit_price: 4000,
      line_total: 8000,
    },
  ],
}

const cancelledOrder: Order = {
  ...createdOrder,
  status: 'CANCELLED',
  cancelled_at: '2026-11-26T10:05:00Z',
}

const queryClients: QueryClient[] = []

function renderOrderDetailPage(
  path = '/orders/1',
  state: unknown = null,
) {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  queryClients.push(queryClient)

  return render(
    <MemoryRouter initialEntries={[{ pathname: path, state }]}>
      <QueryClientProvider client={queryClient}>
        <Routes>
          <Route path="/orders/:id" element={<OrderDetailPage />} />
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

describe('OrderDetailPage', () => {
  it('renders the order total from the server', async () => {
    vi.spyOn(api, 'getOrder').mockResolvedValue(createdOrder)

    renderOrderDetailPage()

    expect(await screen.findByText('Rp 8.000')).toBeInTheDocument()
  })

  it('confirms cancellation and shows that stock was restored', async () => {
    vi.spyOn(api, 'getOrder').mockResolvedValue(createdOrder)
    const cancelOrder = vi
      .spyOn(api, 'cancelOrder')
      .mockResolvedValue(cancelledOrder)
    const user = userEvent.setup()

    renderOrderDetailPage()

    await user.click(await screen.findByRole('button', { name: 'Cancel order' }))
    expect(screen.getByRole('dialog')).toHaveTextContent(
      'Cancel order #1? Stock will be restored.',
    )
    await user.click(screen.getByRole('button', { name: 'Yes, cancel order' }))

    expect(cancelOrder).toHaveBeenCalledWith(1)
    expect(
      await screen.findByText('Order cancelled — stock restored'),
    ).toBeInTheDocument()
    expect(await screen.findByText('CANCELLED')).toBeInTheDocument()
  })

  it('shows a 409 cancellation error in an alert', async () => {
    vi.spyOn(api, 'getOrder').mockResolvedValue(createdOrder)
    vi.spyOn(api, 'cancelOrder').mockRejectedValue(
      new ApiError(
        409,
        'ORDER_ALREADY_CANCELLED',
        'order is already cancelled',
      ),
    )
    const user = userEvent.setup()

    renderOrderDetailPage()

    await user.click(await screen.findByRole('button', { name: 'Cancel order' }))
    await user.click(screen.getByRole('button', { name: 'Yes, cancel order' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'order is already cancelled',
    )
  })

  it('does not show Cancel order for an already cancelled order', async () => {
    vi.spyOn(api, 'getOrder').mockResolvedValue(cancelledOrder)

    renderOrderDetailPage()

    await screen.findByText('CANCELLED')
    expect(
      screen.queryByRole('button', { name: 'Cancel order' }),
    ).not.toBeInTheDocument()
  })

  it('shows a created banner when navigation state says justCreated', async () => {
    vi.spyOn(api, 'getOrder').mockResolvedValue(createdOrder)

    renderOrderDetailPage('/orders/1', { justCreated: true })

    expect(await screen.findByText('Order #1 created')).toBeInTheDocument()
  })

  it('shows a not-found link for a non-numeric order id without fetching', async () => {
    const getOrder = vi.spyOn(api, 'getOrder').mockResolvedValue(createdOrder)

    renderOrderDetailPage('/orders/not-a-number')

    expect(await screen.findByText('Order not found')).toBeInTheDocument()
    expect(getOrder).not.toHaveBeenCalled()
    expect(screen.getByRole('link', { name: /orders/i })).toHaveAttribute(
      'href',
      '/orders',
    )
  })

  it('shows a not-found link when the order endpoint returns 404', async () => {
    vi.spyOn(api, 'getOrder').mockRejectedValue(
      new ApiError(404, 'NOT_FOUND', 'Order not found'),
    )

    renderOrderDetailPage()

    expect(await screen.findByText('Order not found')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /orders/i })).toHaveAttribute(
      'href',
      '/orders',
    )
  })
})
