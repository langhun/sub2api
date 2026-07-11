import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useGameHallStore } from '@/stores/gameHall'
import { exchangeGameBalance, getGameHallStatus, playGame } from '@/api/gameHall'

vi.mock('@/api/gameHall', () => ({
  createGameHallIdempotencyKey: vi.fn((scope: string) => `${scope}-stable-key`),
  getGameHallStatus: vi.fn(),
  exchangeGameBalance: vi.fn(),
  playGame: vi.fn(),
}))

describe('game hall store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('loads balances and enabled games', async () => {
    vi.mocked(getGameHallStatus).mockResolvedValue({
      main_balance: 120,
      dg_balance: 30,
      jackpot_balance: 500,
      games: [{
        type: 'slots', name: 'Slots', min_bet: 1, max_bet: 10, multipliers: [0, 2],
        rule_version: 'slots-v1', theoretical_rtp: 0.953,
        payout_rules: [{ symbol: 'cherry', match_count: 3, multiplier: 18.7, probability: 0.0166 }],
      }],
    })

    const store = useGameHallStore()
    await store.refresh()

    expect(store.status?.dg_balance).toBe(30)
    expect(store.enabledGames).toHaveLength(1)
    expect(store.error).toBe('')
  })

  it('applies server-confirmed balances after an exchange', async () => {
    vi.mocked(exchangeGameBalance).mockResolvedValue({ direction: 'balance_to_dg', amount: 10, main_balance_before: 100, main_balance_after: 90, dg_balance_before: 0, dg_balance_after: 10 })

    const store = useGameHallStore()
    store.status = { main_balance: 100, dg_balance: 0, jackpot_balance: 0, games: [] }
    await store.exchange('balance_to_dg', 10)

    expect(exchangeGameBalance).toHaveBeenCalledWith('balance_to_dg', 10, 'game-exchange-stable-key')
    expect(store.status?.main_balance).toBe(90)
    expect(store.status?.dg_balance).toBe(10)
    expect(getGameHallStatus).not.toHaveBeenCalled()
  })

  it('applies a server-settled round without client-side randomization', async () => {
    vi.mocked(playGame).mockResolvedValue({ game_type: 'slots', bet_amount: 2, payout_amount: 6, net_amount: 4, multiplier: 3, dg_balance_before: 10, dg_balance_after: 14, jackpot_balance: 96, outcome: 'win', symbols: ['STAR', 'STAR', 'STAR'], message: 'win' })
    const store = useGameHallStore()
    store.status = { main_balance: 100, dg_balance: 10, jackpot_balance: 100, games: [] }

    await store.play('slots', 2)

    expect(playGame).toHaveBeenCalledWith('slots', 2, 'game-play-stable-key')
    expect(store.lastRound?.symbols).toEqual(['STAR', 'STAR', 'STAR'])
    expect(store.status.dg_balance).toBe(14)
    expect(store.status.jackpot_balance).toBe(96)
  })

  it('reuses an exchange operation key after an ambiguous failure', async () => {
    vi.mocked(exchangeGameBalance).mockRejectedValue(new Error('timeout'))
    const store = useGameHallStore()

    await expect(store.exchange('balance_to_dg', 10)).rejects.toThrow('timeout')
    await expect(store.exchange('balance_to_dg', 10)).rejects.toThrow('timeout')

    expect(vi.mocked(exchangeGameBalance).mock.calls.map((call) => call[2])).toEqual([
      'game-exchange-stable-key',
      'game-exchange-stable-key',
    ])
  })

  it('reuses a play operation key after an ambiguous failure', async () => {
    vi.mocked(playGame).mockRejectedValue(new Error('timeout'))
    const store = useGameHallStore()

    await expect(store.play('slots', 2)).rejects.toThrow('timeout')
    await expect(store.play('slots', 2)).rejects.toThrow('timeout')

    expect(vi.mocked(playGame).mock.calls.map((call) => call[2])).toEqual([
      'game-play-stable-key',
      'game-play-stable-key',
    ])
  })
})
