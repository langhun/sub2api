import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { getPublicPricing } from '@/api/pricing'

describe('pricing api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('requests public pricing from the public pricing endpoint and returns response data', async () => {
    const data = {
      groups: [
        {
          id: 1,
          name: 'Default',
          platform: 'openai',
          rate_multiplier: 1.2,
          models: [
            {
              model_name: 'gpt-5.1',
              input_cost_per_million: 1.5,
              output_cost_per_million: 6,
              effective_input: 1.8,
              effective_output: 7.2,
            },
          ],
        },
      ],
    }
    get.mockResolvedValue({ data })

    const result = await getPublicPricing()

    expect(get).toHaveBeenCalledWith('/public/pricing')
    expect(result).toEqual(data)
  })
})
