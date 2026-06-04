import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SlotsView from '../SlotsView.vue'

const { getHall, playGame, showError, showSuccess } = vi.hoisted(() => ({
  getHall: vi.fn(),
  playGame: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
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
    showSuccess,
  }),
}))

vi.mock('@/utils/format', () => ({
  formatDualDisplayAmount: (value: number) => ({
    display: String(value),
    full: String(value),
  }),
}))

const gameHall = {
  balance: 888,
  games: [
    {
      type: 'slots',
      name: 'Slots',
      description: 'Three reels with instant settlement.',
      min_bet: 0.01,
      max_bet: 100000000,
      multipliers: [0, 1.2, 3, 5, 8, 12, 18, 30, 50],
    },
  ],
}

const slotsResult = {
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

function mountPage() {
  return mount(SlotsView, {
    global: {
      stubs: {
        AppLayout: {
          template: '<div data-testid="app-layout"><slot /></div>',
        },
        RouterLink: {
          props: ['to'],
          template: '<a :href="to"><slot /></a>',
        },
      },
    },
  })
}

describe('SlotsView', () => {
  beforeEach(() => {
    getHall.mockReset()
    playGame.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    getHall.mockResolvedValue(gameHall)
    playGame.mockResolvedValue(slotsResult)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('loads slot status and renders a standalone slot machine page', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('老虎机')
    expect(wrapper.text()).toContain('娱乐余额')
    expect(wrapper.text()).toContain('888 DG')
    expect(wrapper.findAll('.slot-reel')).toHaveLength(3)
    expect(wrapper.get('[data-testid="play-slots"]').text()).toContain('拉杆')
    expect(getHall).toHaveBeenCalledTimes(1)
  })

  it('enters a rolling state before settling to the backend result', async () => {
    vi.useFakeTimers()

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="slot-bet-amount"]').setValue('25')
    await wrapper.get('[data-testid="play-slots"]').trigger('click')

    expect(playGame).toHaveBeenCalledWith('slots', 25)
    expect(wrapper.get('[data-testid="play-slots"]').text()).toContain('滚动中')
    expect(wrapper.get('[data-testid="slot-reel-0"]').classes()).toContain('is-spinning')

    await vi.advanceTimersByTimeAsync(900)
    await flushPromises()
    await flushPromises()

    expect(wrapper.get('[data-testid="slots-result"]').text()).toContain('已中奖')
    expect(wrapper.get('[data-testid="slots-result"]').text()).toContain('返奖：75 DG')
    expect(wrapper.get('[data-testid="slot-reel-0"]').text()).toContain('樱桃')
    expect(wrapper.get('[data-testid="play-slots"]').text()).toContain('拉杆')
    expect(wrapper.get('[data-testid="slot-reel-0"]').classes()).not.toContain('is-spinning')
    expect(showSuccess).toHaveBeenCalledWith('Win: payout is 3x')
    expect(getHall).toHaveBeenCalledTimes(2)
  })

  it('validates bet amount before allowing play', async () => {
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="slot-bet-amount"]').setValue('0')
    expect(wrapper.get('[data-testid="play-slots"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('最低投注 0.01 DG。')

    await wrapper.get('[data-testid="slot-bet-amount"]').setValue('25')
    expect(wrapper.get('[data-testid="play-slots"]').attributes('disabled')).toBeUndefined()
  })

  it('shows a readable error when play fails', async () => {
    vi.useFakeTimers()
    playGame.mockRejectedValue({ reason: 'BANK_INSUFFICIENT_BALANCE' })

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="play-slots"]').trigger('click')
    await vi.advanceTimersByTimeAsync(900)
    await flushPromises()

    expect(wrapper.text()).toContain('DG 币余额不足，无法开始游戏。')
    expect(showError).toHaveBeenCalledWith('DG 币余额不足，无法开始游戏。')
  })
})
