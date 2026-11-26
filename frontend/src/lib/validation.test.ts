import { describe, expect, it } from 'vitest'

import {
  hasErrors,
  normalizeSku,
  parseQty,
  QTY_MESSAGE,
  toCreateOrderInput,
  validateProductForm,
  validateOrderDraft,
  validateStockAdjust,
} from './validation'
import type {
  ProductForm,
  ProductFormErrors,
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

describe('validateProductForm', () => {
  const validForm: ProductForm = {
    sku: 'BRS-1',
    name: 'Beras',
    price: '0',
    stock: '0',
    isActive: true,
  }

  it.each<{
    name: string
    form: ProductForm
    mode: 'create' | 'edit'
    expected: ProductFormErrors
  }>([
    {
      name: 'requires SKU when creating',
      form: { ...validForm, sku: ' ' },
      mode: 'create',
      expected: { sku: 'is required' },
    },
    {
      name: 'limits SKU length to 64 characters',
      form: { ...validForm, sku: 'A'.repeat(65) },
      mode: 'create',
      expected: { sku: 'must be at most 64 characters' },
    },
    {
      name: 'requires a nonblank name',
      form: { ...validForm, name: '   ' },
      mode: 'create',
      expected: { name: 'is required' },
    },
    {
      name: 'limits name length to 200 characters',
      form: { ...validForm, name: 'N'.repeat(201) },
      mode: 'create',
      expected: { name: 'must be at most 200 characters' },
    },
    {
      name: 'requires price',
      form: { ...validForm, price: '' },
      mode: 'create',
      expected: { price: 'is required' },
    },
    {
      name: 'limits price to 1000000000',
      form: { ...validForm, price: '1000000001' },
      mode: 'create',
      expected: { price: 'must be between 0 and 1000000000' },
    },
    {
      name: 'does not allow negative stock',
      form: { ...validForm, stock: '-1' },
      mode: 'create',
      expected: { stock: 'must be between 0 and 1000000' },
    },
    {
      name: 'allows an omitted optional stock value',
      form: { ...validForm, stock: '' },
      mode: 'create',
      expected: {},
    },
    {
      name: 'ignores SKU and stock in edit mode',
      form: { ...validForm, sku: '', stock: '-1' },
      mode: 'edit',
      expected: {},
    },
  ])('$name', ({ form, mode, expected }) => {
    expect(validateProductForm(form, mode)).toEqual(expected)
  })

  it('rejects SKU with invalid characters', () => {
    expect(
      validateProductForm({ ...validForm, sku: 'AB C' }, 'create'),
    ).toEqual({
      sku: 'may contain only A-Z, 0-9 and -',
    })
  })

  it('rejects a price below zero', () => {
    expect(
      validateProductForm({ ...validForm, price: '-1' }, 'create'),
    ).toEqual({
      price: 'must be between 0 and 1000000000',
    })
  })

  it('rejects stock above the allowed range', () => {
    expect(
      validateProductForm({ ...validForm, stock: '1000001' }, 'create'),
    ).toEqual({
      stock: 'must be between 0 and 1000000',
    })
  })

  it('accepts a valid product form', () => {
    expect(validateProductForm(validForm, 'create')).toEqual({})
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