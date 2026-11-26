export function formatRupiah(n: number): string {
  return `Rp ${new Intl.NumberFormat('id-ID', {
    maximumFractionDigits: 0,
  }).format(n)}`
}

export function timeAgo(iso: string | null, now = new Date()): string {
  if (iso === null) {
    return '—'
  }

  const then = new Date(iso)
  const diff = now.getTime() - then.getTime()

  if (diff < 60_000) {
    return 'just now'
  }

  if (diff < 60 * 60_000) {
    return `${Math.floor(diff / 60_000)}m ago`
  }

  if (diff < 24 * 60 * 60_000) {
    return `${Math.floor(diff / (60 * 60_000))}h ago`
  }

  if (diff < 30 * 24 * 60 * 60_000) {
    return `${Math.floor(diff / (24 * 60 * 60_000))}d ago`
  }

  return then.toISOString().slice(0, 10)
}