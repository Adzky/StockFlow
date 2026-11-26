import { describeError } from '../lib/errors'

export function LoadingState({ label = 'Loading...' }: { label?: string }) {
  return (
    <div className="state-view" role="status">
      {label}
    </div>
  )
}

export function EmptyState({
  message,
  action,
}: {
  message: string
  action?: { label: string; onClick: () => void }
}) {
  return (
    <div className="state-view">
      <p>{message}</p>
      {action && (
        <button type="button" onClick={action.onClick}>
          {action.label}
        </button>
      )}
    </div>
  )
}

export function ErrorState({
  error,
  onRetry,
}: {
  error: unknown
  onRetry: () => void
}) {
  return (
    <div className="alert" role="alert">
      <p>{describeError(error)}</p>
      <button type="button" onClick={onRetry}>
        Retry
      </button>
    </div>
  )
}