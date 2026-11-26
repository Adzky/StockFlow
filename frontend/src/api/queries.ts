import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from '@tanstack/react-query'
import type { QueryClient } from '@tanstack/react-query'

import { api } from './client'
import type {
  CreateOrderInput,
  CreateProductInput,
  ProductQuery,
  StockAdjustInput,
  UpdateProductInput,
} from './types'

export const qk = {
  products: (query: ProductQuery) => ['products', query] as const,
  summary: ['summary'] as const,
  movements: (id: number) => ['movements', id] as const,
  orders: (page: number) => ['orders', page] as const,
  order: (id: number) => ['order', id] as const,
}

function invalidateStock(queryClient: QueryClient) {
  return Promise.all([
    queryClient.invalidateQueries({ queryKey: ['products'] }),
    queryClient.invalidateQueries({ queryKey: qk.summary }),
  ])
}

export function useProducts(query: ProductQuery) {
  return useQuery({
    queryKey: qk.products(query),
    queryFn: () => api.listProducts(query),
    placeholderData: keepPreviousData,
  })
}

export function useSummary() {
  return useQuery({
    queryKey: qk.summary,
    queryFn: api.getSummary,
  })
}

export function useMovements(id: number) {
  return useQuery({
    queryKey: qk.movements(id),
    queryFn: () => api.listMovements(id),
  })
}

export function useOrders(page: number) {
  return useQuery({
    queryKey: qk.orders(page),
    queryFn: () => api.listOrders(page, 20),
    placeholderData: keepPreviousData,
  })
}

export function useOrder(id: number) {
  return useQuery({
    queryKey: qk.order(id),
    queryFn: () => api.getOrder(id),
    enabled: Number.isInteger(id) && id > 0,
  })
}

export function useAdjustStock() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: number
      input: StockAdjustInput
    }) => api.adjustStock(id, input),
    onSuccess: (_result, { id }) =>
      Promise.all([
        invalidateStock(queryClient),
        queryClient.invalidateQueries({
          queryKey: qk.movements(id),
        }),
      ]),
  })
}

export function useCreateOrder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateOrderInput) => api.createOrder(input),
    onSuccess: (order) => {
      queryClient.setQueryData(qk.order(order.id), order)

      return Promise.all([
        invalidateStock(queryClient),
        queryClient.invalidateQueries({ queryKey: ['orders'] }),
      ])
    },
  })
}

export function useCancelOrder() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => api.cancelOrder(id),
    onSuccess: (order) => {
      queryClient.setQueryData(qk.order(order.id), order)

      return Promise.all([
        invalidateStock(queryClient),
        queryClient.invalidateQueries({ queryKey: ['orders'] }),
      ])
    },
  })
}

export function useCreateProduct() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (input: CreateProductInput) => api.createProduct(input),
    onSuccess: () => invalidateStock(queryClient),
  })
}

export function useUpdateProduct() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: ({
      id,
      input,
    }: {
      id: number
      input: UpdateProductInput
    }) => api.updateProduct(id, input),
    onSuccess: () => invalidateStock(queryClient),
  })
}

export function useDeleteProduct() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => api.deleteProduct(id),
    onSuccess: () => invalidateStock(queryClient),
  })
}