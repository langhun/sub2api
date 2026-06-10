import { flushPromises, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GameHallView from '../GameHallView.vue'

const { getHall, showError } = vi.hoisted(() => ({
  getHall: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/games', () => ({
  gamesAPI: {
    getHall,
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

function mountPage() {
  return mount(GameHallView, {
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

describe('GameHallView', () => {
  beforeEach(() => {
    getHall.mockReset()
    showError.mockReset()
    getHall.mockResolvedValue(hallStatus)
  })

  it('renders DG balance and jackpot in the hall summary', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const hallSummary = wrapper.get('[data-testid="hall-summary"]')

    expect(wrapper.text()).toContain('娱乐大厅')
    expect(hallSummary.text()).toContain('DG 余额')
    expect(hallSummary.text()).toContain('大厅奖池')
    expect(hallSummary.text()).toContain('12 DG')
    expect(hallSummary.text()).toContain('345 DG')
    expect(hallSummary.text()).not.toContain('主余额')
    expect(hallSummary.text()).not.toContain('88')
    expect(getHall).toHaveBeenCalledTimes(1)
  })

  it('shows main balance only inside the exchange card', async () => {
    const wrapper = mountPage()
    await flushPromises()
    const exchangeCard = wrapper.get('[data-testid="exchange-card"]')

    expect(exchangeCard.text()).toContain('余额兑换')
    expect(exchangeCard.text()).toContain('主余额')
    expect(exchangeCard.text()).toContain('88')
    expect(exchangeCard.text()).toContain('DG 币')
    expect(exchangeCard.text()).toContain('12 DG')
  })

  it('renders a slots entry inside the hall', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('幸运老虎机')
    expect(wrapper.text()).toContain('进入游戏')
    expect(wrapper.text()).toContain('当前可用范围：0.01 DG - 100000000 DG')
    expect(wrapper.find('a[href="/games/slots"]').exists()).toBe(true)
  })

  it('shows a readable error when hall loading fails', async () => {
    getHall.mockRejectedValue(new Error('hall api failed'))

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('hall api failed')
    expect(showError).toHaveBeenCalledWith('hall api failed')
  })

  it('shows chinese amount hints for large exchange values', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const amountInput = wrapper.get('input[type="number"]')
    const directionSelect = wrapper.get('select')

    await amountInput.setValue('10000000')
    await nextTick()
    expect(wrapper.get('[data-testid="exchange-amount-hint"]').text()).toBe('1千万DG币')
    expect(wrapper.get('[data-testid="exchange-amount-helper"]').text()).toBe('按 1:1 兑换，预计到账 1千万DG币')

    await amountInput.setValue('12345678')
    await nextTick()
    expect(wrapper.get('[data-testid="exchange-amount-hint"]').text()).toBe('1.235千万DG币')

    await amountInput.setValue('100000000')
    await nextTick()
    expect(wrapper.get('[data-testid="exchange-amount-hint"]').text()).toBe('1亿DG币')

    await amountInput.setValue('123456789')
    await nextTick()
    expect(wrapper.get('[data-testid="exchange-amount-hint"]').text()).toBe('1.235亿DG币')

    await directionSelect.setValue('dg_to_balance')
    await nextTick()
    expect(wrapper.get('[data-testid="exchange-amount-hint"]').text()).toBe('1.235亿主余额')
    expect(wrapper.get('[data-testid="exchange-amount-helper"]').text()).toBe('按 1:1 兑换，预计到账 1.235亿主余额')
  })
})
