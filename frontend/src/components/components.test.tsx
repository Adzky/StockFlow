import { fireEvent, render, screen } from '@testing-library/react'
import {
  QueryClient,
  QueryClientProvider,
} from '@tanstack/react-query'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { api, ApiError } from '../api/client'
import type { Summary } from '../api/types'
import { describeError } from '../lib/errors'
import { ErrorState } from './StateViews'
import { SummaryCards } from './SummaryCards'

afterEach(() => {
  vi.restoreAllMocks()
})

describe('shared components', () => {
  it('ErrorState shows the API error and calls Retry when clicked', () => {
    const error = new ApiError(
      0,
      'NETWORK_ERROR',
      'Cannot reach API',
    )
    const onRetry = vi.fn()

    render(<ErrorState error={error} onRetry={onRetry} />)

    expect(screen.getByRole('alert')).toHaveTextContent(
      describeError(error),
    )

    fireEvent.click(screen.getByRole('button', { name: 'Retry' }))

    expect(onRetry).toHaveBeenCalledTimes(1)
  })

  it('SummaryCards shows the seeded stock value and low-stock count', async () => {
    const summary: Summary = {
      total_products: 15,
      active_products: 13,
      total_stock_value: 15_069_000,
      low_stock_count: 4,
    }

    vi.spyOn(api, 'getSummary').mockResolvedValue(summary)

    const queryClient = new QueryClient({
      defaultOptions: {
        queries: { retry: false },
      },
    })

    render(
      <QueryClientProvider client={queryClient}>
        <SummaryCards />
      </QueryClientProvider>,
    )

    expect(await screen.findByText('Rp 15.069.000')).toBeInTheDocument()
    expect(await screen.findByText('4')).toBeInTheDocument()

    queryClient.clear()
  })
})