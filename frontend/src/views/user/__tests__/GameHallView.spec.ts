import { flushPromises, mount } from '@vue/test-utils'
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

  it('renders main balance, DG balance, jackpot and slots entry', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('娱乐大厅')
    expect(wrapper.text()).toContain('主余额')
    expect(wrapper.text()).toContain('DG 余额')
    expect(wrapper.text()).toContain('大厅奖池')
    expect(wrapper.text()).toContain('88')
    expect(wrapper.text()).toContain('12 DG')
    expect(wrapper.text()).toContain('345 DG')
    expect(wrapper.get('[data-testid="slots-entry"]').attributes('href')).toBe('/games/slots')
    expect(getHall).toHaveBeenCalledTimes(1)
  })

  it('shows a readable error when hall loading fails', async () => {
    getHall.mockRejectedValue(new Error('hall api failed'))

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('hall api failed')
    expect(wrapper.get('[data-testid="slots-entry"]').attributes('href')).toBe('/games/slots')
    expect(showError).toHaveBeenCalledWith('hall api failed')
  })
})
