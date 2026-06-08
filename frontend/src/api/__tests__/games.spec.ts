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

import { gamesAPI } from '@/api/games'

describe('games api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    vi.stubGlobal('crypto', {
      randomUUID: () => 'game-uuid-123',
    } as unknown as Crypto)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads the game hall status', async () => {
    const hall = {
      main_balance: 88,
      dg_balance: 12,
      jackpot_balance: 345,
      games: [
        {
          type: 'slots',
          name: 'Slots',
          description: 'Three reels with instant DG settlement.',
          min_bet: 0.01,
          max_bet: 1000000,
          multipliers: [0, 1.2, 3, 5, 8, 12, 18, 30, 50],
        },
      ],
    }
    get.mockResolvedValue({ data: hall })

    await expect(gamesAPI.getHall()).resolves.toEqual(hall)
    expect(get).toHaveBeenCalledWith('/games/hall')
  })

  it('exchanges balance and DG with an explicit idempotency key', async () => {
    const result = {
      direction: 'balance_to_dg',
      amount: 20,
      main_balance_before: 80,
      main_balance_after: 60,
      dg_balance_before: 5,
      dg_balance_after: 25,
    }
    post.mockResolvedValue({ data: result })

    await expect(gamesAPI.exchange('balance_to_dg', 20, 'exchange-key-1')).resolves.toEqual(result)
    expect(post).toHaveBeenCalledWith('/games/exchange', {
      direction: 'balance_to_dg',
      amount: 20,
    }, {
      headers: {
        'Idempotency-Key': 'exchange-key-1',
      },
    })
  })

  it('plays slots with a generated idempotency key', async () => {
    const result = {
      game_type: 'slots',
      bet_amount: 10,
      payout_amount: 0,
      net_amount: -10,
      multiplier: 0,
      dg_balance_before: 12,
      dg_balance_after: 2,
      jackpot_balance: 355,
      outcome: 'lose',
      symbols: ['cherry', 'lemon', 'bell'],
      message: '未中奖',
    }
    post.mockResolvedValue({ data: result })

    await expect(gamesAPI.play('slots', 10)).resolves.toEqual(result)
    expect(post).toHaveBeenCalledWith('/games/play', {
      game_type: 'slots',
      bet_amount: 10,
    }, {
      headers: {
        'Idempotency-Key': 'game:slots:10:game-uuid-123',
      },
    })
  })
})
