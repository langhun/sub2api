import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest'
import GameSlotsView from '../GameSlotsView.vue'

const { getHall, play, showError } = vi.hoisted(() => ({
  getHall: vi.fn(),
  play: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/games', () => ({
  gamesAPI: {
    getHall,
    play,
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
  dg_balance: 120,
  jackpot_balance: 345,
  games: [
    {
      type: 'slots',
      name: 'Slots',
      description: 'Three reels with instant DG settlement.',
      min_bet: 10,
      max_bet: 1000000,
      multipliers: [0, 1.2, 3, 5, 8, 12, 18, 30, 50],
    },
  ],
}

const totalSpinSettleMs = 2600
let rafTimestamp = 0

function mountPage() {
  return mount(GameSlotsView, {
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

describe('GameSlotsView', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    rafTimestamp = performance.now()
    vi.spyOn(window, 'requestAnimationFrame').mockImplementation((callback: FrameRequestCallback) => {
      return window.setTimeout(() => {
        rafTimestamp += 16
        callback(rafTimestamp)
      }, 16) as unknown as number
    })
    vi.spyOn(window, 'cancelAnimationFrame').mockImplementation((id: number) => {
      window.clearTimeout(id)
    })
    getHall.mockReset()
    play.mockReset()
    showError.mockReset()
    getHall.mockResolvedValue(hallStatus)
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  it('renders the slot machine page with DG-only balance information', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('幸运老虎机')
    expect(wrapper.text()).toContain('返回娱乐大厅')
    expect(wrapper.text()).toContain('DG 币')
    expect(wrapper.text()).toContain('胜率')
    expect(wrapper.text()).toContain('本场盈亏')
    expect(wrapper.text()).toContain('120')
    expect(wrapper.text()).not.toContain('120 DG')
    expect(wrapper.text()).toContain('自定义 DG 投注')
    expect(wrapper.text()).toContain('5 连抽')
    expect(wrapper.text()).toContain('10 连抽')
    expect(wrapper.text()).toContain('10')
    expect(wrapper.text()).toContain('100')
    expect(wrapper.text()).toContain('1000')
    expect(wrapper.text()).toContain('1万')
    expect(wrapper.text()).toContain('10万')
    expect(wrapper.text()).toContain('100万')
    expect(wrapper.text()).not.toContain('主余额')
    expect(wrapper.text()).not.toContain('DG 余额')
    expect(wrapper.text()).not.toContain('奖池')
    expect(wrapper.text()).not.toContain('345 DG')
    expect(wrapper.text()).not.toContain('左边保留机台沉浸感')
    expect(wrapper.text()).not.toContain('本局结果')
    expect(wrapper.text()).not.toContain('等待结果')
    expect(wrapper.get('.machine-panel').text()).not.toContain('幸运老虎机')
  })

  it('keeps the payout table collapsed by default and expands it on demand', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.get('[data-testid="slots-odds-toggle"]').attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('#slots-odds-table').exists()).toBe(false)

    await wrapper.get('[data-testid="slots-odds-toggle"]').trigger('click')

    expect(wrapper.get('[data-testid="slots-odds-toggle"]').attributes('aria-expanded')).toBe('true')
    expect(wrapper.find('#slots-odds-table').exists()).toBe(true)
    expect(wrapper.text()).toContain('50x')
    expect(wrapper.text()).toContain('1.2x')
  })

  it('plays slots and updates balances from the API result', async () => {
    play.mockResolvedValue({
      game_type: 'slots',
      bet_amount: 20,
      payout_amount: 60,
      net_amount: 40,
      multiplier: 3,
      dg_balance_before: 120,
      dg_balance_after: 160,
      jackpot_balance: 305,
      outcome: 'win',
      symbols: ['cherry', 'cherry', 'cherry'],
      message: '中奖',
    })

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="slots-bet-100"]').trigger('click')
    await wrapper.get('[data-testid="slots-spin-button"]').trigger('click')
    await vi.advanceTimersByTimeAsync(totalSpinSettleMs)
    await flushPromises()

    expect(play).toHaveBeenCalledWith('slots', 100)
    expect(wrapper.text()).toContain('160')
    expect(wrapper.text()).not.toContain('160 DG')
    expect(wrapper.text()).not.toContain('305 DG')
    expect(wrapper.text()).toContain('+40')
  })

  it('uses the custom DG bet for real sequential 5-spin settlement', async () => {
    let callCount = 0
    play.mockImplementation(() => {
      callCount += 1
      return Promise.resolve({
        game_type: 'slots',
        bet_amount: 75,
        payout_amount: 0,
        net_amount: -75,
        multiplier: 0,
        dg_balance_before: 120 - ((callCount - 1) * 75),
        dg_balance_after: 120 - (callCount * 75),
        jackpot_balance: 345 + (callCount * 75),
        outcome: 'lose',
        symbols: ['cherry', 'lemon', 'orange'],
        message: '未中奖',
      })
    })

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="slots-custom-bet"]').setValue('75')
    await wrapper.get('[data-testid="slots-spin-5-button"]').trigger('click')

    for (let index = 0; index < 5; index += 1) {
      await vi.advanceTimersByTimeAsync(totalSpinSettleMs)
      await flushPromises()
    }

    expect(play).toHaveBeenCalledTimes(5)
    expect(play).toHaveBeenNthCalledWith(1, 'slots', 75)
    expect(play).toHaveBeenNthCalledWith(5, 'slots', 75)
    expect(wrapper.text()).toContain('-255')
    expect(wrapper.text()).not.toContain('-255 DG')
    expect(wrapper.findAll('.history-table__row')).toHaveLength(5)
  })

  it('keeps accumulating session net after more than 10 spins and makes history scrollable', async () => {
    let callCount = 0
    play.mockImplementation(() => {
      callCount += 1
      return Promise.resolve({
        game_type: 'slots',
        bet_amount: 10,
        payout_amount: 0,
        net_amount: -10,
        multiplier: 0,
        dg_balance_before: 120 - ((callCount - 1) * 10),
        dg_balance_after: 120 - (callCount * 10),
        jackpot_balance: 345 + (callCount * 10),
        outcome: 'lose',
        symbols: ['cherry', 'lemon', 'orange'],
        message: '未中奖',
      })
    })

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="slots-spin-10-button"]').trigger('click')
    for (let index = 0; index < 10; index += 1) {
      await vi.advanceTimersByTimeAsync(totalSpinSettleMs)
      await flushPromises()
    }

    await wrapper.get('[data-testid="slots-spin-button"]').trigger('click')
    await vi.advanceTimersByTimeAsync(totalSpinSettleMs)
    await flushPromises()

    expect(play).toHaveBeenCalledTimes(11)
    expect(wrapper.get('[data-testid="slots-session-net"]').text()).toBe('-110')
    expect(wrapper.get('[data-testid="slots-history-count"]').text()).toBe('11')
    expect(wrapper.findAll('.history-table__row')).toHaveLength(11)
    expect(wrapper.get('[data-testid="slots-history-table"]').classes()).toContain('history-table--scrollable')
  })

  it('shows a Chinese amount hint for custom bet input', async () => {
    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="slots-custom-bet"]').setValue('1000000')

    expect(wrapper.get('[data-testid="slots-custom-bet-hint"]').text()).toBe('100万')
    expect(wrapper.get('[data-testid="slots-custom-bet"]').attributes('placeholder')).toContain('最高 1亿')
  })

  it('shows reel strip animation state while a spin is in progress', async () => {
    let resolvePlay: ((value: {
      game_type: 'slots'
      bet_amount: number
      payout_amount: number
      net_amount: number
      multiplier: number
      dg_balance_before: number
      dg_balance_after: number
      jackpot_balance: number
      outcome: 'win'
      symbols: string[]
      message: string
    }) => void) | null = null

    play.mockReturnValue(new Promise((resolve) => {
      resolvePlay = resolve
    }))

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="slots-spin-button"]').trigger('click')
    await flushPromises()

    expect(wrapper.findAll('[data-testid^="slots-reel-strip-"]')).toHaveLength(3)
    expect(wrapper.findAll('.slot-machine__strip--spinning')).toHaveLength(3)

    resolvePlay?.({
      game_type: 'slots',
      bet_amount: 20,
      payout_amount: 60,
      net_amount: 40,
      multiplier: 3,
      dg_balance_before: 120,
      dg_balance_after: 160,
      jackpot_balance: 305,
      outcome: 'win',
      symbols: ['cherry', 'cherry', 'cherry'],
      message: '中奖',
    })
    await vi.advanceTimersByTimeAsync(totalSpinSettleMs)
    await flushPromises()
  })

  it('recycles reel strip content while spinning', async () => {
    let resolvePlay: ((value: {
      game_type: 'slots'
      bet_amount: number
      payout_amount: number
      net_amount: number
      multiplier: number
      dg_balance_before: number
      dg_balance_after: number
      jackpot_balance: number
      outcome: 'win'
      symbols: string[]
      message: string
    }) => void) | null = null

    play.mockReturnValue(new Promise((resolve) => {
      resolvePlay = resolve
    }))

    const wrapper = mountPage()
    await flushPromises()

    await wrapper.get('[data-testid="slots-spin-button"]').trigger('click')
    await flushPromises()

    const startText = wrapper.get('[data-testid="slots-reel-strip-0"]').text()
    const startTransform = wrapper.get('[data-testid="slots-reel-strip-0"]').attributes('style')

    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()

    expect(wrapper.get('[data-testid="slots-reel-strip-0"]').attributes('style')).not.toBe(startTransform)
    expect(wrapper.get('[data-testid="slots-reel-strip-0"]').text()).not.toBe(startText)

    resolvePlay?.({
      game_type: 'slots',
      bet_amount: 20,
      payout_amount: 60,
      net_amount: 40,
      multiplier: 3,
      dg_balance_before: 120,
      dg_balance_after: 160,
      jackpot_balance: 305,
      outcome: 'win',
      symbols: ['cherry', 'cherry', 'cherry'],
      message: '中奖',
    })
    await vi.advanceTimersByTimeAsync(totalSpinSettleMs)
    await flushPromises()
  })

  it('keeps the configured fixed tiers even when backend max bet is smaller', async () => {
    getHall.mockResolvedValue({
      ...hallStatus,
      games: [
        {
          ...hallStatus.games[0],
          min_bet: 5,
          max_bet: 15,
        },
      ],
    })
    play.mockResolvedValue({
      game_type: 'slots',
      bet_amount: 10,
      payout_amount: 0,
      net_amount: -15,
      multiplier: 0,
      dg_balance_before: 120,
      dg_balance_after: 105,
      jackpot_balance: 360,
      outcome: 'lose',
      symbols: ['cherry', 'lemon', 'orange'],
      message: '未中奖',
    })

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.get('[data-testid="slots-bet-10"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="slots-bet-100"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="slots-bet-1000"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="slots-bet-1000000"]').exists()).toBe(true)

    await wrapper.get('[data-testid="slots-spin-button"]').trigger('click')
    await vi.advanceTimersByTimeAsync(totalSpinSettleMs)
    await flushPromises()

    expect(play).toHaveBeenCalledWith('slots', 10)
  })
})
