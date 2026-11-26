import { useState } from 'react'

import { ApiError } from '../api/client'
import { useDeleteProduct, useProducts } from '../api/queries'
import type { Product, ProductFilter } from '../api/types'
import { Modal } from '../components/Modal'
import { Pagination } from '../components/Pagination'
import { ProductFormModal } from '../components/ProductFormModal'
import { ProductTable } from '../components/ProductTable'
import { EmptyState, ErrorState, LoadingState } from '../components/StateViews'
import { SummaryCards } from '../components/SummaryCards'
import { AdjustStockModal } from '../components/AdjustStockModal'
import { useDebounce } from '../hooks/useDebounce'
import { describeError } from '../lib/errors'

type ProductFormState =
  | { mode: 'create' }
  | { mode: 'edit'; product: Product }

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
  const [productForm, setProductForm] = useState<ProductFormState | null>(null)
  const [deletingProduct, setDeletingProduct] = useState<Product | null>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const deleteProduct = useDeleteProduct()

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

  function openDeleteConfirmation(product: Product) {
    deleteProduct.reset()
    setDeleteError(null)
    setDeletingProduct(product)
  }

  function closeDeleteConfirmation() {
    setDeletingProduct(null)
    setDeleteError(null)
  }

  function confirmDelete() {
    if (!deletingProduct) {
      return
    }

    deleteProduct.mutate(deletingProduct.id, {
      onSuccess: () => {
        setFlashMessage(`${deletingProduct.sku} deleted`)
        closeDeleteConfirmation()
      },
      onError: (error) => {
        if (
          error instanceof ApiError
          && error.status === 409
          && error.code === 'PRODUCT_HAS_HISTORY'
        ) {
          setDeleteError(
            "This product has stock history and can't be deleted. Deactivate it instead.",
          )
          return
        }

        setDeleteError(describeError(error))
      },
    })
  }

  return (
    <main className="products-page">
      <SummaryCards />

      <h1>Products</h1>
      <button
        type="button"
        onClick={() => setProductForm({ mode: 'create' })}
      >
        New product
      </button>

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
            onEdit={(product) => {
              setProductForm({ mode: 'edit', product })
            }}
            onDelete={openDeleteConfirmation}
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

      {productForm?.mode === 'create' && (
        <ProductFormModal
          mode="create"
          onClose={() => setProductForm(null)}
        />
      )}

      {productForm?.mode === 'edit' && (
        <ProductFormModal
          mode="edit"
          product={productForm.product}
          onClose={() => setProductForm(null)}
        />
      )}

      {deletingProduct && (
        <Modal
          title={`Delete ${deletingProduct.sku}?`}
          onClose={closeDeleteConfirmation}
        >
          <p>Delete {deletingProduct.name}? This action cannot be undone.</p>
          {deleteError && <p role="alert">{deleteError}</p>}
          <button
            type="button"
            disabled={deleteProduct.isPending}
            onClick={confirmDelete}
          >
            Confirm delete
          </button>
          <button
            type="button"
            disabled={deleteProduct.isPending}
            onClick={closeDeleteConfirmation}
          >
            Cancel
          </button>
        </Modal>
      )}
    </main>
  )
}