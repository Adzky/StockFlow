export function Pagination({
  page,
  limit,
  total,
  onPage,
}: {
  page: number
  limit: number
  total: number
  onPage: (page: number) => void
}) {
  const totalPages = Math.max(1, Math.ceil(total / limit))

  return (
    <nav className="pagination" aria-label="Product pagination">
      <button
        type="button"
        disabled={page <= 1}
        onClick={() => onPage(page - 1)}
      >
        Prev
      </button>
      <span aria-live="polite">
        Page {page} of {totalPages}
      </span>
      <button
        type="button"
        disabled={page >= totalPages}
        onClick={() => onPage(page + 1)}
      >
        Next
      </button>
    </nav>
  )
}