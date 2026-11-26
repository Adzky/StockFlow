import { describe, expect, it } from 'vitest'

import { formatRupiah, timeAgo } from './format'

describe('formatRupiah', () => {
  it.each([
    [1234567, 'Rp 1.234.567'],
    [0, 'Rp 0'],
    [15069000, 'Rp 15.069.000'],
  ])('formats %i as %s', (value, expected) => {
    expect(formatRupiah(value)).toBe(expected)
  })
})

describe('timeAgo', () => {
  const now = new Date('2026-11-25T12:00:00Z')

  it.each([
    [null, '—'],
    ['2026-11-25T11:59:30Z', 'just now'],
    ['2026-11-25T12:03:00Z', 'just now'],
    ['2026-11-25T11:55:00Z', '5m ago'],
    ['2026-11-25T10:00:00Z', '2h ago'],
    ['2026-11-22T12:00:00Z', '3d ago'],
    ['2026-10-11T12:00:00Z', '2026-10-11'],
  ])('formats %s as %s', (iso, expected) => {
    expect(timeAgo(iso, now)).toBe(expected)
  })
})