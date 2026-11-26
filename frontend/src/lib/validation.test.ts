import { describe, expect, it } from 'vitest'

import {
  hasErrors,
  normalizeSku,
  parseQty,
  QTY_MESSAGE,
  toCreateOrderInput,
  validateOrderDraft,
  validateStockAdjust,
} from './validation'

describe('parseQty', () => {
  it.each([
    ['1', 1],
    ['10', 10],
    [' 5 ', 5],
    ['1000000', 1_000_000],
    ['1000001', null],
    ['1.5', null],
    ['0', null],
    ['-1', null],
    ['abc', null],
    ['', null],
    ['1e3', null],
  ])('parses %j as %s', (raw, expected) => {
    expect(parseQty(raw)).toBe(expected)
  })
})

describe('normalizeSku', () => {
  it('trims whitespace and converts to uppercase', () => {
    expect(normalizeSku(' brs-1 ')).toBe('BRS-1')
  })
})

describe('validateStockAdjust', () => {
  it('returns a quantity error for invalid quantity', () => {
    expect(
      validateStockAdjust({ type: 'in', qty: '1.5', note: '' }),
    ).toEqual({ qty: QTY_MESSAGE })
  })

  it('returns a note error when note exceeds 500 characters', () => {
    expect(
      validateStockAdjust({
        type: 'in',
        qty: '1',
        note: 'a'.repeat(501),
      }),
    ).toEqual({ note: 'Note must be at most 500 characters' })
  })

  it('returns no errors for valid input', () => {
    expect(
      validateStockAdjust({ type: 'out', qty: '3', note: 'sale' }),
    ).toEqual({})
  })
})

describe('validateOrderDraft', () => {
  it('reports a blank customer and missing items', () => {
    expect(
      validateOrderDraft({ customer: ' ', items: [] }),
    ).toEqual({
      customer: 'Customer name is required',
      items: {},
      general: ['Add at least one item'],
    })
  })

  it('marks a duplicate normalized SKU on the later row', () => {
    expect(
      validateOrderDraft({
        customer: 'Toko',
        items: [
          { sku: 'abc', qty: '1' },
          { sku: ' ABC ', qty: '2' },
        ],
      }),
    ).toEqual({
      items: { 1: 'Duplicate SKU — already in row 1' },
      general: [],
    })
  })

  it('prioritizes the blank SKU error over invalid quantity', () => {
    expect(
      validateOrderDraft({
        customer: 'Toko',
        items: [{ sku: '', qty: '0' }],
      }),
    ).toEqual({
      items: { 0: 'SKU is required' },
      general: [],
    })
  })
})

describe('hasErrors', () => {
  it('returns false when there are no stock adjustment errors', () => {
    expect(hasErrors({})).toBe(false)
  })

  it('returns true when there are order form errors', () => {
    expect(
      hasErrors({
        items: { 0: 'SKU is required' },
        general: [],
      }),
    ).toBe(true)
  })
})

describe('toCreateOrderInput', () => {
  it('trims customer, normalizes SKUs, and parses quantities', () => {
    expect(
      toCreateOrderInput({
        customer: ' Toko ',
        items: [{ sku: ' brs-1 ', qty: '2' }],
      }),
    ).toEqual({
      customer: 'Toko',
      items: [{ sku: 'BRS-1', qty: 2 }],
    })
  })
})