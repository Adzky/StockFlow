import { useSummary } from '../api/queries'
import { formatRupiah } from '../lib/format'
import { ErrorState, LoadingState } from './StateViews'

export function SummaryCards() {
  const { data, error, isPending, refetch } = useSummary()

  if (isPending) {
    return <LoadingState label="Loading summary..." />
  }

  if (error) {
    return (
      <ErrorState
        error={error}
        onRetry={() => {
          void refetch()
        }}
      />
    )
  }

  if (!data) {
    return null
  }

  return (
    <section className="summary-cards" aria-label="Summary">
      <article className="summary-card">
        <h2>Total products</h2>
        <p>{data.total_products}</p>
      </article>
      <article className="summary-card">
        <h2>Active products</h2>
        <p>{data.active_products}</p>
      </article>
      <article className="summary-card">
        <h2>Stock value</h2>
        <p>{formatRupiah(data.total_stock_value)}</p>
      </article>
      <article className="summary-card">
        <h2>Low stock (≤ 10)</h2>
        <p>{data.low_stock_count}</p>
      </article>
    </section>
  )
}