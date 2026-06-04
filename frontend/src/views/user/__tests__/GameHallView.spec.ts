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
    getHall.mockResolvedValue(gameHall)
  })

  it('renders the entertainment hall as two independent game entries', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.text()).toContain('娱乐大厅')
    expect(wrapper.text()).toContain('老虎机')
    expect(wrapper.text()).toContain('双色球')
    expect(wrapper.text()).toContain('888 DG')
    expect(wrapper.get('[data-testid="slots-entry"]').attributes('href')).toBe('/games/slots')
    expect(wrapper.get('[data-testid="ssq-entry"]').attributes('href')).toBe('/games/ssq')
    expect(wrapper.find('[data-testid="red-ball-grid"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="play-slots"]').exists()).toBe(false)
    expect(getHall).toHaveBeenCalledTimes(1)
  })

  it('keeps both game entries reachable when hall status loading fails', async () => {
    getHall.mockRejectedValue(new Error('hall api failed'))

    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.get('[data-testid="slots-entry"]').attributes('href')).toBe('/games/slots')
    expect(wrapper.get('[data-testid="ssq-entry"]').attributes('href')).toBe('/games/ssq')
    expect(wrapper.text()).toContain('hall api failed')
    expect(showError).toHaveBeenCalledWith('hall api failed')
  })
})
