import { beforeEach, describe, expect, it, vi } from 'vitest'

const { put } = vi.hoisted(() => ({ put: vi.fn() }))

vi.mock('@/api/client', () => ({ apiClient: { put } }))

import { updateSmartRouting, type SmartRoutingUpdate } from '../api'

const payload: SmartRoutingUpdate = {
  version: 2,
  enabled: true,
  mode: 'smart',
  strategy: 'auto',
  price_weight: 40,
  speed_weight: 30,
  success_weight: 30,
  sticky_ttl_seconds: 0,
  exploration_rate: 0.05,
  group_ids: [21]
}

describe('smart routing API', () => {
  beforeEach(() => {
    put.mockReset()
    put.mockResolvedValue({ data: { api_key_id: 9 } })
  })

  it('caps legacy exploration rates without mutating the caller payload', async () => {
    await updateSmartRouting(9, payload)

    expect(put).toHaveBeenCalledWith('/keys/9/routing', {
      ...payload,
      exploration_rate: 0.01
    })
    expect(payload.exploration_rate).toBe(0.05)
  })

  it('preserves exploration rates already within the server limit', async () => {
    await updateSmartRouting(9, { ...payload, exploration_rate: 0.005 })

    expect(put).toHaveBeenCalledWith('/keys/9/routing', expect.objectContaining({ exploration_rate: 0.005 }))
  })
})
