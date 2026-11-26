import { useState } from 'react'
import { Link } from 'react-router-dom'

import { useOrders } from '../api/queries'
import { Pagination } from '../components/Pagination'
import { EmptyState, ErrorState, LoadingState } from '../components/StateViews'
import { formatRupiah, timeAgo } from '../lib/format'

export function OrdersPage() {
  const [page, setPage] = useState(1)
  const ordersQuery = useOrders(page)

  return (
    <main className="orders-page">
      <h1>Orders</h1>

      {ordersQuery.isPending && <LoadingState label="Loading orders..." />}

      {ordersQuery.isError && (
        <ErrorState
          error={ordersQuery.error}
          onRetry={() => {
            void ordersQuery.refetch()
          }}
        />
      )}

      {ordersQuery.isSuccess && ordersQuery.data.data.length === 0 && (
        <EmptyState message="No orders yet" />
      )}

      {ordersQuery.isSuccess && ordersQuery.data.data.length > 0 && (
        <>
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th scope="col">Order</th>
                  <th scope="col">Customer</th>
                  <th scope="col">Status</th>
                  <th scope="col">Items</th>
                  <th scope="col">Total</th>
                  <th scope="col">Created</th>
                </tr>
              </thead>
              <tbody>
                {ordersQuery.data.data.map((order) => (
                  <tr key={order.id}>
                    <td>
                      <Link to={`/orders/${order.id}`}>#{order.id}</Link>
                    </td>
                    <td>{order.customer}</td>
                    <td>
                      <span
                        className={`status-badge ${
                          order.status === 'CREATED'
                            ? 'status-badge--active'
                            : 'status-badge--inactive'
                        }`}
                      >
                        {order.status}
                      </span>
                    </td>
                    <td>{order.item_count}</td>
                    <td>{formatRupiah(order.total)}</td>
                    <td>
                      <time
                        dateTime={order.created_at}
                        title={new Date(order.created_at).toLocaleString()}
                      >
                        {timeAgo(order.created_at)}
                      </time>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <Pagination
            page={page}
            limit={ordersQuery.data.limit}
            total={ordersQuery.data.total}
            onPage={setPage}
          />
        </>
      )}
    </main>
  )
}
