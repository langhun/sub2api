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
      randomUUID: () => 'uuid-123',
    } as unknown as Crypto)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads game hall status', async () => {
    const hall = {
      balance: 888,
      games: [
        {
          type: 'slots',
          name: 'Slots',
          description: 'Three reels with instant settlement.',
          min_bet: 0.01,
          max_bet: 100000000,
          multipliers: [0, 1.2, 3],
        },
      ],
    }
    get.mockResolvedValue({ data: hall })

    await expect(gamesAPI.getHall()).resolves.toEqual(hall)
    expect(get).toHaveBeenCalledWith('/games/hall')
  })

  it('plays slots with an idempotency key', async () => {
    const result = {
      game_type: 'slots',
      bet_amount: 25,
      payout_amount: 75,
      net_amount: 50,
      multiplier: 3,
      balance_before: 888,
      balance_after: 938,
      outcome: 'win',
      symbols: ['cherry', 'cherry', 'cherry'],
      message: 'Win: payout is 3x',
    }
    post.mockResolvedValue({ data: result })

    await expect(gamesAPI.play('slots', 25)).resolves.toEqual(result)

    expect(post).toHaveBeenCalledWith('/games/play', {
      game_type: 'slots',
      bet_amount: 25,
    }, {
      headers: {
        'Idempotency-Key': 'game:slots:25:uuid-123',
      },
    })
  })
})
