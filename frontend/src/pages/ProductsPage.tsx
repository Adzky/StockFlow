import { useState } from 'react'

import { useProducts } from '../api/queries'
import type { Product, ProductFilter } from '../api/types'
import { ProductTable } from '../components/ProductTable'
import { Pagination } from '../components/Pagination'
import { EmptyState, ErrorState, LoadingState } from '../components/StateViews'
import { SummaryCards } from '../components/SummaryCards'
import { AdjustStockModal } from '../components/AdjustStockModal'
import { useDebounce } from '../hooks/useDebounce'

const filters: { value: ProductFilter; label: string }[] = [
  { value: 'all', label: 'All' },
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
  { value: 'low_stock', label: 'Low stock' },
]

export function ProductsPage() {
  const [search, setSearch] = useState('')
  const [filter, setFilter] = useState<ProductFilter>('all')
  const [page, setPage] = useState(1)
  const debouncedSearch = useDebounce(search, 300)
  const [adjustingProduct, setAdjustingProduct] = useState<Product | null>(null)
  const [flashMessage, setFlashMessage] = useState<string | null>(null)

  const {
    data,
    error,
    isError,
    isPending,
    isSuccess,
    refetch,
  } = useProducts({
    search: debouncedSearch,
    filter,
    page,
    limit: 20,
  })

  function clearFilters() {
    setSearch('')
    setFilter('all')
    setPage(1)
  }

  return (
    <main className="products-page">
      <SummaryCards />

      <h1>Products</h1>

      <div className="products-toolbar">
        <label htmlFor="product-search">Search products</label>
        <input
          id="product-search"
          type="search"
          value={search}
          onChange={(event) => {
            setSearch(event.target.value)
            setPage(1)
          }}
        />

        <div role="group" aria-label="Product filters">
          {filters.map((option) => (
            <button
              key={option.value}
              type="button"
              aria-pressed={filter === option.value}
              onClick={() => {
                setFilter(option.value)
                setPage(1)
              }}
            >
              {option.label}
            </button>
          ))}
        </div>
      </div>

      {isPending && <LoadingState label="Loading products..." />}

      {isError && (
        <ErrorState
          error={error}
          onRetry={() => {
            void refetch()
          }}
        />
      )}

      {isSuccess && data.data.length === 0 && (
        <EmptyState
          message="No products match"
          action={{ label: 'Clear filters', onClick: clearFilters }}
        />
      )}

      {isSuccess && data.data.length > 0 && (
        <>
          <ProductTable
            products={data.data}
            onAdjust={(product) => {
              setFlashMessage(null)
              setAdjustingProduct(product)
            }}
          />
          <Pagination
            page={page}
            limit={20}
            total={data.total}
            onPage={setPage}
          />
        </>
      )}

      {flashMessage && <p role="status">{flashMessage}</p>}

      {adjustingProduct && (
        <AdjustStockModal
          product={adjustingProduct}
          onClose={() => setAdjustingProduct(null)}
          onSaved={(result) => {
            const before =
              result.movement.type === 'in'
                ? result.product.stock - result.movement.qty
                : result.product.stock + result.movement.qty

            setFlashMessage(
              `${result.product.sku}: stock ${before} → ${result.product.stock}`,
            )
          }}
        />
      )}
    </main>
  )
}