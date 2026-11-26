import { useState } from 'react'
import { Link, useLocation, useParams } from 'react-router-dom'

import { ApiError } from '../api/client'
import { useCancelOrder, useOrder } from '../api/queries'
import { Modal } from '../components/Modal'
import { ErrorState, LoadingState } from '../components/StateViews'
import { describeError } from '../lib/errors'
import { formatRupiah, timeAgo } from '../lib/format'

function OrderNotFound() {
  return (
    <main>
      <h1>Order not found</h1>
      <Link to="/orders">Back to orders</Link>
    </main>
  )
}

export function OrderDetailPage() {
  const { id = '' } = useParams()
  const location = useLocation()
  const orderId = Number(id)
  const validId = /^\d+$/.test(id) && Number.isSafeInteger(orderId) && orderId > 0
  const orderQuery = useOrder(validId ? orderId : 0)
  const cancelMutation = useCancelOrder()
  const [showCancelConfirmation, setShowCancelConfirmation] = useState(false)

  const locationState: unknown = location.state
  const justCreated =
    typeof locationState === 'object'
    && locationState !== null
    && 'justCreated' in locationState
    && locationState.justCreated === true

  if (!validId) {
    return <OrderNotFound />
  }

  if (orderQuery.isPending) {
    return <LoadingState label="Loading order..." />
  }

  if (orderQuery.isError) {
    if (
      orderQuery.error instanceof ApiError
      && orderQuery.error.status === 404
    ) {
      return <OrderNotFound />
    }

    return (
      <main>
        <ErrorState
          error={orderQuery.error}
          onRetry={() => {
            void orderQuery.refetch()
          }}
        />
        <Link to="/orders">Back to orders</Link>
      </main>
    )
  }

  const order = orderQuery.data

  function closeCancelConfirmation() {
    setShowCancelConfirmation(false)
    cancelMutation.reset()
  }

  return (
    <main className="order-detail-page">
      {justCreated && (
        <p role="status">
          Order #{order.id} created
        </p>
      )}

      <header>
        <h1>Order #{order.id}</h1>
        <p>{order.customer}</p>
        <span
          className={`status-badge ${
            order.status === 'CREATED'
              ? 'status-badge--active'
              : 'status-badge--inactive'
          }`}
        >
          {order.status}
        </span>
        <p>
          Created:{' '}
          <time
            dateTime={order.created_at}
            title={new Date(order.created_at).toLocaleString()}
          >
            {timeAgo(order.created_at)}
          </time>
        </p>
        <p>
          Cancelled:{' '}
          {order.cancelled_at ? (
            <time
              dateTime={order.cancelled_at}
              title={new Date(order.cancelled_at).toLocaleString()}
            >
              {timeAgo(order.cancelled_at)}
            </time>
          ) : (
            '—'
          )}
        </p>
      </header>

      {cancelMutation.isSuccess && (
        <p role="status">Order cancelled — stock restored</p>
      )}

      <div className="table-wrap">
        <table>
          <thead>
            <tr>
              <th scope="col">SKU</th>
              <th scope="col">Name</th>
              <th scope="col">Qty</th>
              <th scope="col">Unit price</th>
              <th scope="col">Line total</th>
            </tr>
          </thead>
          <tbody>
            {order.items.map((item) => (
              <tr key={item.id}>
                <td>{item.sku}</td>
                <td>{item.name}</td>
                <td>{item.qty}</td>
                <td>{formatRupiah(item.unit_price)}</td>
                <td>{formatRupiah(item.line_total)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <p>
        <strong>Total: {formatRupiah(order.total)}</strong>
      </p>

      {order.status === 'CREATED' && (
        <button
          type="button"
          onClick={() => setShowCancelConfirmation(true)}
        >
          Cancel order
        </button>
      )}

      <p>
        <Link to="/">Back to Products</Link>
      </p>

      {showCancelConfirmation && (
        <Modal
          title={`Cancel order #${order.id}? Stock will be restored.`}
          onClose={closeCancelConfirmation}
        >
          <p>Cancel this order and restore its stock?</p>
          {cancelMutation.isError && (
            <div role="alert">{describeError(cancelMutation.error)}</div>
          )}
          <button
            type="button"
            disabled={cancelMutation.isPending}
            onClick={() =>
              cancelMutation.mutate(order.id, {
                onSuccess: () => setShowCancelConfirmation(false),
              })
            }
          >
            Yes, cancel order
          </button>
          <button
            type="button"
            disabled={cancelMutation.isPending}
            onClick={closeCancelConfirmation}
          >
            Keep order
          </button>
        </Modal>
      )}
    </main>
  )
}
