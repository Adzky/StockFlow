import { describe, expect, it } from 'vitest'

import { ApiError } from '../api/client'
import type { OrderDraft } from './validation'
import {
  describeError,
  insufficientStockMessage,
  mapOrderError,
} from './errors'

const draft: OrderDraft = {
  customer: 'Toko',
  items: [
    { sku: 'A', qty: '1' },
    { sku: ' B ', qty: '2' },
  ],
}

describe('describeError', () => {
  it('returns the message from an ApiError', () => {
    expect(
      describeError(new ApiError(500, 'UNKNOWN', 'Server error')),
    ).toBe('Server error')
  })

  it('returns a fallback for other errors', () => {
    expect(describeError(new Error('unexpected'))).toBe(
      'Something went wrong',
    )
  })
})

describe('insufficientStockMessage', () => {
  it('includes the available quantity, including zero', () => {
    expect(insufficientStockMessage(0)).toBe(
      'Insufficient stock — only 0 available',
    )
  })
})

describe('mapOrderError', () => {
  it('maps insufficient stock to the matching SKU row', () => {
    const error = new ApiError(
      409,
      'INSUFFICIENT_STOCK',
      'insufficient stock',
      { sku: 'B', available: 0 },
    )

    expect(mapOrderError(error, draft)).toEqual({
      items: { 1: 'Only 0 available' },
      general: [],
    })
  })

  it('puts unmatched insufficient stock in general errors', () => {
    const error = new ApiError(
      409,
      'INSUFFICIENT_STOCK',
      'insufficient stock',
      { sku: 'MISSING', available: 0 },
    )

    expect(mapOrderError(error, draft)).toEqual({
      items: {},
      general: ['Insufficient stock — only 0 available'],
    })
  })

  it('maps validation details to rows and customer', () => {
    const error = new ApiError(400, 'VALIDATION_FAILED', 'validation failed', {
      details: [
        { field: 'items[1].sku', message: 'unknown sku "NOPE"' },
        { field: 'items[1].qty', message: 'quantity is invalid' },
        { field: 'customer', message: 'is required' },
        { field: 'items', message: 'order items are invalid' },
        { field: 'other', message: 'another error' },
      ],
    })

    expect(mapOrderError(error, draft)).toEqual({
      customer: 'is required',
      items: { 1: 'unknown sku "NOPE"' },
      general: ['order items are invalid', 'another error'],
    })
  })

  it('uses the ApiError message for other API errors', () => {
    const error = new ApiError(500, 'UNKNOWN', 'Server error')

    expect(mapOrderError(error, draft)).toEqual({
      items: {},
      general: ['Server error'],
    })
  })

  it('uses a fallback for non-ApiError values', () => {
    expect(mapOrderError(new Error('unexpected'), draft)).toEqual({
      items: {},
      general: ['Something went wrong'],
    })
  })
})