import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import { lotteryAPI } from '@/api/lottery'

describe('lottery api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    vi.stubGlobal('crypto', {
      randomUUID: () => 'uuid-123',
    } as unknown as Crypto)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads current issue from lottery current endpoint', async () => {
    const current = {
      lottery_type: 'ssq',
      issue_no: '2026060',
      open_time: '2026-06-04T13:15:00Z',
      cutoff_time: '2026-06-04T13:05:00Z',
      is_closed: false,
      jackpot_balance: '10000070',
    }
    get.mockResolvedValue({ data: current })

    await expect(lotteryAPI.getCurrent()).resolves.toEqual(current)
    expect(get).toHaveBeenCalledWith('/lottery/current')
  })

  it('creates bet with idempotency key based on issue and numbers', async () => {
    const result = {
      order_id: 42,
      issue_no: '2026060',
      lottery_type: 'ssq',
      cost: '100',
      status: 'pending',
      created_at: '2026-06-04T12:00:00Z',
    }
    const payload = {
      red_balls: [33, 1, 12, 8, 25, 18],
      blue_ball: 9,
    }
    post.mockResolvedValue({ data: result })

    await expect(lotteryAPI.createBet(payload, '2026060')).resolves.toEqual(result)

    expect(post).toHaveBeenCalledWith('/lottery/bet', payload, {
      headers: {
        'Idempotency-Key': 'lottery:2026060:1-8-12-18-25-33:9:uuid-123',
      },
    })
  })

  it('loads orders with optional issue filter', async () => {
    const orders = [
      {
        order_id: 1,
        lottery_type: 'ssq',
        issue_no: '2026060',
        red_balls: ['01', '08', '12', '18', '25', '33'],
        blue_ball: '09',
        cost: '100',
        status: 'pending',
        reward: '0',
        prize_level: '',
        created_at: '2026-06-04T12:00:00Z',
      },
    ]
    get.mockResolvedValue({ data: orders })

    await expect(lotteryAPI.getOrders('2026060')).resolves.toEqual(orders)
    expect(get).toHaveBeenCalledWith('/lottery/orders', {
      params: { issue_no: '2026060' },
    })

    await lotteryAPI.getOrders()
    expect(get).toHaveBeenLastCalledWith('/lottery/orders', {
      params: undefined,
    })
  })
})
