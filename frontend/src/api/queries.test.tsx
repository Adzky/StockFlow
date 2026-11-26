import type { PropsWithChildren } from 'react'
import { act, renderHook } from '@testing-library/react'
import {
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api } from './client'
import type {
  Order,
  Product,
  StockAdjustResult,
} from './types'
import { useAdjustStock, useCancelOrder } from './queries'

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: PropsWithChildren) {
    return (
      <QueryClientProvider client={queryClient}>
        {children}
      </QueryClientProvider>
    )
  }
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('order and stock mutation hooks', () => {
  it('cancelOrder invalidates stock-related queries and caches the order', async () => {
    const queryClient = new QueryClient()
    const order: Order = {
      id: 1,
      customer: 'Toko',
      status: 'CANCELLED',
      total: 2500,
      created_at: '2026-01-01T00:00:00Z',
      cancelled_at: '2026-01-01T00:05:00Z',
      items: [],
    }

    vi.spyOn(api, 'cancelOrder').mockResolvedValue(order)
    const invalidateSpy = vi
      .spyOn(queryClient, 'invalidateQueries')
      .mockResolvedValue(undefined)

    const { result } = renderHook(() => useCancelOrder(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync(1)
    })

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ['products'],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ['summary'],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ['orders'],
    })
    expect(queryClient.getQueryData(['order', 1])).toEqual(order)

    queryClient.clear()
  })

  it('adjustStock invalidates products, summary, and that product’s movements', async () => {
    const queryClient = new QueryClient()

    const product: Product = {
      id: 7,
      sku: 'BRS-1',
      name: 'Beras',
      price: 1000,
      stock: 8,
      is_active: true,
      created_at: '2026-01-01T00:00:00Z',
      last_movement_at: '2026-01-01T00:05:00Z',
    }

    const resultBody: StockAdjustResult = {
      product,
      movement: {
        id: 12,
        product_id: 7,
        type: 'in',
        qty: 3,
        note: 'restock',
        order_id: null,
        created_at: '2026-01-01T00:05:00Z',
      },
    }

    vi.spyOn(api, 'adjustStock').mockResolvedValue(resultBody)
    const invalidateSpy = vi
      .spyOn(queryClient, 'invalidateQueries')
      .mockResolvedValue(undefined)

    const { result } = renderHook(() => useAdjustStock(), {
      wrapper: createWrapper(queryClient),
    })

    await act(async () => {
      await result.current.mutateAsync({
        id: 7,
        input: { type: 'in', qty: 3, note: 'restock' },
      })
    })

    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ['products'],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ['summary'],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ['movements', 7],
    })

    queryClient.clear()
  })
})