import { afterEach, describe, expect, it, vi } from 'vitest'

import { api, ApiError } from './client'

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('API client', () => {
  it('listProducts uses the expected product query URL', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        data: [],
        page: 2,
        limit: 20,
        total: 0,
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await api.listProducts({
      search: 'be ras',
      filter: 'low_stock',
      page: 2,
      limit: 20,
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/products?search=be+ras&filter=low_stock&page=2&limit=20',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('omits an empty product search', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        data: [],
        page: 1,
        limit: 20,
        total: 0,
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await api.listProducts({
      search: '',
      filter: 'all',
      page: 1,
      limit: 20,
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/products?filter=all&page=1&limit=20',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('createOrder sends POST JSON', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        id: 1,
        customer: 'Budi',
        status: 'CREATED',
        total: 0,
        created_at: '2026-01-01T00:00:00Z',
        cancelled_at: null,
        items: [],
      }),
    )
    vi.stubGlobal('fetch', fetchMock)

    await api.createOrder({
      customer: 'Budi',
      items: [{ sku: 'A', qty: 2 }],
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/orders',
      expect.objectContaining({
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          customer: 'Budi',
          items: [{ sku: 'A', qty: 2 }],
        }),
      }),
    )
  })

  it('parses insufficient-stock errors', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse(
          {
            error: {
              code: 'INSUFFICIENT_STOCK',
              message: 'insufficient stock',
              sku: 'B',
              available: 1,
            },
          },
          409,
        ),
      ),
    )

    await expect(
      api.createOrder({
        customer: 'Budi',
        items: [{ sku: 'B', qty: 5 }],
      }),
    ).rejects.toMatchObject({
      status: 409,
      code: 'INSUFFICIENT_STOCK',
      sku: 'B',
      available: 1,
    })
  })

  it('preserves available zero', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse(
          {
            error: {
              code: 'INSUFFICIENT_STOCK',
              message: 'insufficient stock',
              sku: 'A',
              available: 0,
            },
          },
          409,
        ),
      ),
    )

    try {
      await api.createOrder({
        customer: 'Budi',
        items: [{ sku: 'A', qty: 1 }],
      })
      expect.fail('Expected API call to reject')
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError)
      expect(error).toMatchObject({
        status: 409,
        code: 'INSUFFICIENT_STOCK',
        available: 0,
      })
    }
  })

  it('preserves validation details', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        jsonResponse(
          {
            error: {
              code: 'VALIDATION_FAILED',
              message: 'validation failed',
              details: [
                { field: 'customer', message: 'is required' },
              ],
            },
          },
          400,
        ),
      ),
    )

    try {
      await api.createOrder({
        customer: '',
        items: [],
      })
      expect.fail('Expected API call to reject')
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError)
      expect(error).toMatchObject({
        status: 400,
        code: 'VALIDATION_FAILED',
        details: [
          { field: 'customer', message: 'is required' },
        ],
      })
    }
  })

  it('maps fetch throws to NETWORK_ERROR', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockRejectedValue(new TypeError('Failed to fetch')),
    )

    try {
      await api.getSummary()
      expect.fail('Expected API call to reject')
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError)
      expect(error).toMatchObject({
        status: 0,
        code: 'NETWORK_ERROR',
        message: 'Cannot reach the server. Is the backend running?',
      })
    }
  })

  it('maps plain 502 response to UNKNOWN', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response('Bad Gateway', { status: 502 }),
      ),
    )

    try {
      await api.getSummary()
      expect.fail('Expected API call to reject')
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError)
      expect(error).toMatchObject({
        status: 502,
        code: 'UNKNOWN',
        message: 'Server error (HTTP 502)',
      })
    }
  })

  it('deleteProduct resolves undefined for 204', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
    )

    await expect(api.deleteProduct(1)).resolves.toBeUndefined()
  })
})