import { beforeEach, describe, expect, it, vi } from 'vitest'

const { del, get, post, put } = vi.hoisted(() => ({
  del: vi.fn(),
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: { delete: del, get, post, put },
}))

import {
  createPrizeItem,
  deletePrizeItem,
  getBlindboxStats,
  listPrizeItems,
  updatePrizeItem,
} from '@/custom/modules/activity/api/admin/blindbox'
import {
  compensateRewardDelivery,
  listRewardDeliveries,
  retryRewardDelivery,
} from '@/custom/modules/activity/api/admin/rewardDeliveries'

describe('activity admin APIs', () => {
  beforeEach(() => {
    del.mockReset().mockResolvedValue({})
    get.mockReset().mockResolvedValue({ data: {} })
    post.mockReset().mockResolvedValue({ data: {} })
    put.mockReset().mockResolvedValue({ data: {} })
  })

  it('keeps blindbox prize management scoped to the activity module endpoints', async () => {
    const prize = { name: 'bonus', rarity: 'common', reward_type: 'balance' }

    await listPrizeItems()
    await createPrizeItem(prize)
    await updatePrizeItem(12, { weight: 80 })
    await deletePrizeItem(12)
    await getBlindboxStats()

    expect(get).toHaveBeenNthCalledWith(1, '/admin/blindbox/prize-items')
    expect(post).toHaveBeenCalledWith('/admin/blindbox/prize-items', prize)
    expect(put).toHaveBeenCalledWith('/admin/blindbox/prize-items/12', { weight: 80 })
    expect(del).toHaveBeenCalledWith('/admin/blindbox/prize-items/12')
    expect(get).toHaveBeenNthCalledWith(2, '/admin/blindbox/stats')
  })

  it('keeps reward delivery operations attached to the activity blindbox source', async () => {
    const params = { status: 'failed' as const, source_type: 'checkin_blindbox', page: 2, page_size: 10 }

    await listRewardDeliveries(params)
    await retryRewardDelivery(9)
    await compensateRewardDelivery(9, 'manual credit')

    expect(get).toHaveBeenCalledWith('/admin/reward-deliveries', { params })
    expect(post).toHaveBeenNthCalledWith(1, '/admin/reward-deliveries/9/retry')
    expect(post).toHaveBeenNthCalledWith(2, '/admin/reward-deliveries/9/compensate', { reason: 'manual credit' })
  })
})
