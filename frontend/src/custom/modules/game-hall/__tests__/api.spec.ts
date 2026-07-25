import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({ apiClient: { get, post } }))

import {
  exchangeGameBalance,
  getGameHallStatus,
  getGameRounds,
  getGameTransactions,
  playGame,
} from '../api'

describe('game hall API', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    get.mockResolvedValue({ data: {} })
    post.mockResolvedValue({ data: {} })
  })

  it('uses the module status and history endpoints', async () => {
    await getGameHallStatus()
    await getGameTransactions(2, 25)
    await getGameRounds(3, 50)

    expect(get).toHaveBeenNthCalledWith(1, '/game-hall/status')
    expect(get).toHaveBeenNthCalledWith(2, '/game-hall/transactions', { params: { page: 2, page_size: 25 } })
    expect(get).toHaveBeenNthCalledWith(3, '/game-hall/rounds', { params: { page: 3, page_size: 50 } })
  })

  it('uses module mutation endpoints with caller-supplied idempotency keys', async () => {
    await exchangeGameBalance('balance_to_dg', 12.5, 'exchange-request-key')
    await playGame('slots', 3, 'play-request-key')

    expect(post).toHaveBeenNthCalledWith(
      1,
      '/game-hall/exchange',
      { direction: 'balance_to_dg', amount: 12.5 },
      { headers: { 'Idempotency-Key': 'exchange-request-key' } },
    )
    expect(post).toHaveBeenNthCalledWith(
      2,
      '/game-hall/play',
      { game_type: 'slots', bet_amount: 3 },
      { headers: { 'Idempotency-Key': 'play-request-key' } },
    )
  })
})
