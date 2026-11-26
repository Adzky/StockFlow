import type { PropsWithChildren } from 'react'
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
import {
  MemoryRouter,
  Route,
  Routes,
  useParams,
} from 'react-router-dom'

import { api, ApiError } from '../api/client'
import type { Order, Paged, Product } from '../api/types'
import { CreateOrderPage } from './CreateOrderPage'

const product: Product = {
  id: 1,
  sku: 'A',
  name: 'Produk A',
  price: 8000,
  stock: 10,
  is_active: true,
  created_at: '2026-11-25T12:00:00Z',
  last_movement_at: '2026-11-25T12:00:00Z',
}

const productOptions: Paged<Product> = {
  data: [product],
  page: 1,
  limit: 100,
  total: 1,
}

const order: Order = {
  id: 7,
  customer: 'Toko',
  status: 'CREATED',
  total: 8000,
  created_at: '2026-11-26T12:00:00Z',
  cancelled_at: null,
  items: [
    {
      id: 1,
      sku: 'A',
      name: 'Produk A',
      qty: 1,
      unit_price: 8000,
      line_total: 8000,
    },
  ],
}

const queryClients: QueryClient[] = []

function OrderDetailStub() {
  const { id } = useParams()
  return <p>detail {id}</p>
}

function renderCreateOrderPage() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
  queryClients.push(queryClient)

  vi.spyOn(api, 'listProducts').mockResolvedValue(productOptions)

  function Wrapper({ children }: PropsWithChildren) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    )
  }

  return render(
    <MemoryRouter initialEntries={['/orders/new']}>
      <Wrapper>
        <Routes>
          <Route path="/orders/new" element={<CreateOrderPage />} />
          <Route path="/orders/:id" element={<OrderDetailStub />} />
        </Routes>
      </Wrapper>
    </MemoryRouter>,
  )
}

function getIndexedInput(label: string, index: number): HTMLElement {
  const input = screen.getAllByLabelText(label)[index]
  if (!input) {
    throw new Error(`Missing ${label} input at index ${index}`)
  }
  return input
}

async function fillFirstOrderItem(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Customer'), 'Toko')
  await user.type(screen.getByLabelText('SKU'), 'A')
  await user.type(screen.getByLabelText('Qty'), '1')
}

afterEach(() => {
  cleanup()
  queryClients.forEach((queryClient) => queryClient.clear())
  queryClients.length = 0
  vi.restoreAllMocks()
})

describe('CreateOrderPage', () => {
  it('maps a 409 stock error to the matching item row', async () => {
    vi.spyOn(api, 'createOrder').mockRejectedValue(
      new ApiError(409, 'INSUFFICIENT_STOCK', 'insufficient stock', {
        sku: 'B',
        available: 1,
      }),
    )

    const user = userEvent.setup()
    renderCreateOrderPage()

    await user.type(screen.getByLabelText('Customer'), 'Toko')
    await user.type(screen.getByLabelText('SKU'), 'A')
    await user.type(screen.getByLabelText('Qty'), '1')
    await user.click(screen.getByRole('button', { name: 'Add item' }))
    await user.type(getIndexedInput('SKU', 1), 'B')
    await user.type(getIndexedInput('Qty', 1), '5')
    await user.click(screen.getByRole('button', { name: 'Create order' }))

    expect(await screen.findByText('Only 1 available')).toBeInTheDocument()

    const skuInputs = screen.getAllByLabelText('SKU')
    expect(skuInputs[0]).not.toHaveAttribute('aria-invalid', 'true')
    expect(skuInputs[1]).toHaveAttribute('aria-invalid', 'true')
    expect(skuInputs[0]?.closest('.row-error')).toBeNull()
    expect(skuInputs[1]?.closest('.row-error')).toBeInTheDocument()
  })

  it('shows a 400 SKU validation detail on its item row', async () => {
    vi.spyOn(api, 'createOrder').mockRejectedValue(
      new ApiError(400, 'VALIDATION_FAILED', 'validation failed', {
        details: [
          {
            field: 'items[0].sku',
            message: 'unknown sku "NOPE"',
          },
        ],
      }),
    )

    const user = userEvent.setup()
    renderCreateOrderPage()

    await user.type(screen.getByLabelText('Customer'), 'Toko')
    await user.type(screen.getByLabelText('SKU'), 'NOPE')
    await user.type(screen.getByLabelText('Qty'), '1')
    await user.click(screen.getByRole('button', { name: 'Create order' }))

    expect(await screen.findByText('unknown sku "NOPE"')).toBeInTheDocument()
    expect(screen.getByLabelText('SKU')).toHaveAttribute('aria-invalid', 'true')
  })

  it('prevents a second submit while order creation is pending', async () => {
    const createOrder = vi.spyOn(api, 'createOrder').mockImplementation(
      () => new Promise<Order>(() => {}),
    )

    const user = userEvent.setup()
    renderCreateOrderPage()
    await fillFirstOrderItem(user)

    const submitButton = screen.getByRole('button', { name: 'Create order' })
    await user.click(submitButton)
    await waitFor(() => expect(submitButton).toBeDisabled())
    await user.click(submitButton)

    expect(createOrder).toHaveBeenCalledTimes(1)
  })

  it('navigates to the created order detail on success', async () => {
    vi.spyOn(api, 'createOrder').mockResolvedValue(order)

    const user = userEvent.setup()
    renderCreateOrderPage()
    await fillFirstOrderItem(user)
    await user.click(screen.getByRole('button', { name: 'Create order' }))

    expect(await screen.findByText('detail 7')).toBeInTheDocument()
  })

  it('shows customer validation and does not create an order', async () => {
    const createOrder = vi.spyOn(api, 'createOrder').mockResolvedValue(order)

    const user = userEvent.setup()
    renderCreateOrderPage()
    await user.click(screen.getByRole('button', { name: 'Create order' }))

    expect(
      await screen.findByText('Customer name is required'),
    ).toBeInTheDocument()
    expect(createOrder).not.toHaveBeenCalled()
  })
})
