import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import SlotMachinePanel from '../SlotMachinePanel.vue'

const { getHall, playGame, showError } = vi.hoisted(() => ({
  getHall: vi.fn(),
  playGame: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/games', () => ({
  gamesAPI: {
    getHall,
    play: playGame,
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError,
  }),
}))

vi.mock('@/utils/format', () => ({
  formatNumber: (value: number) => String(value),
}))

const hallStatus = {
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

describe('SlotMachinePanel', () => {
  beforeEach(() => {
    getHall.mockReset()
    playGame.mockReset()
    showError.mockReset()
    getHall.mockResolvedValue(hallStatus)
    playGame.mockResolvedValue({
      game_type: 'slots',
      bet_amount: 10,
      payout_amount: 30,
      net_amount: 20,
      multiplier: 3,
      dg_balance_before: 12,
      dg_balance_after: 32,
      jackpot_balance: 325,
      outcome: 'win',
      symbols: ['cherry', 'cherry', 'cherry'],
      message: '中奖',
    })
  })

  it('loads hall status and displays DG balance and jackpot', async () => {
    const wrapper = mount(SlotMachinePanel)
    await flushPromises()

    expect(wrapper.text()).toContain('DG 余额')
    expect(wrapper.text()).toContain('12 DG')
    expect(wrapper.text()).toContain('大厅奖池')
    expect(wrapper.text()).toContain('345 DG')
    expect(getHall).toHaveBeenCalledTimes(1)
  })

  it('plays slots through the games api and updates result text', async () => {
    const wrapper = mount(SlotMachinePanel)
    await flushPromises()

    await wrapper.get('[data-testid="slot-bet-input"]').setValue('10')
    await wrapper.get('[data-testid="slot-spin"]').trigger('click')
    await flushPromises()

    expect(playGame).toHaveBeenCalledWith('slots', 10)
    expect(wrapper.get('[data-testid="slot-result-message"]').text()).toContain('中奖')
    expect(wrapper.text()).toContain('32 DG')
  })
})
