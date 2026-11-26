import type { Product } from '../api/types'
import { formatRupiah, timeAgo } from '../lib/format'
import { StatusBadge } from './StatusBadge'

export function ProductTable({
  products,
  onAdjust,
  onEdit,
  onDelete,
}: {
  products: Product[]
  onAdjust: (product: Product) => void
  onEdit: (product: Product) => void
  onDelete: (product: Product) => void
}) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th scope="col">SKU</th>
            <th scope="col">Name</th>
            <th scope="col">Price</th>
            <th scope="col">Stock</th>
            <th scope="col">Status</th>
            <th scope="col">Last movement</th>
            <th scope="col">Actions</th>
          </tr>
        </thead>
        <tbody>
          {products.map((product) => (
            <tr key={product.id}>
              <td>{product.sku}</td>
              <td>{product.name}</td>
              <td>{formatRupiah(product.price)}</td>
              <td>
                <span className={product.stock <= 10 ? 'low' : undefined}>
                  {product.stock}
                </span>
              </td>
              <td>
                <StatusBadge active={product.is_active} />
              </td>
              <td>
                {product.last_movement_at ? (
                  <time
                    dateTime={product.last_movement_at}
                    title={new Date(product.last_movement_at).toLocaleString()}
                  >
                    {timeAgo(product.last_movement_at)}
                  </time>
                ) : (
                  timeAgo(null)
                )}
              </td>
              <td>
                <button type="button" onClick={() => onAdjust(product)}>
                  Adjust stock
                </button>
                <button type="button" onClick={() => onEdit(product)}>
                  Edit
                </button>
                <button type="button" onClick={() => onDelete(product)}>
                  Delete
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}