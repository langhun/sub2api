import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import LeaderboardView from '@/views/LeaderboardView.vue'
import type { LeaderboardData } from '@/api/leaderboard'

const api = vi.hoisted(() => ({
  balance: vi.fn(),
  consumption: vi.fn(),
  checkin: vi.fn(),
  transfer: vi.fn(),
}))

const appStore = vi.hoisted(() => ({
  publicSettingsLoaded: true,
  cachedPublicSettings: {
    leaderboard_balance_enabled: true,
    leaderboard_consumption_enabled: true,
    leaderboard_checkin_enabled: true,
    leaderboard_transfer_enabled: true,
    transfer_enabled: true,
  } as Record<string, boolean>,
  fetchPublicSettings: vi.fn(),
}))

vi.mock('@/api/leaderboard', () => ({
  leaderboardAPI: {
    getBalanceLeaderboard: api.balance,
    getConsumptionLeaderboard: api.consumption,
    getCheckinLeaderboard: api.checkin,
    getTransferLeaderboard: api.transfer,
  },
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))

vi.mock('chart.js', () => ({
  ArcElement: {},
  Tooltip: {},
  Chart: { register: vi.fn() },
}))

vi.mock('vue-chartjs', () => ({
  Doughnut: defineComponent({ name: 'Doughnut', template: '<div data-testid="doughnut" />' }),
}))

const AppLayoutStub = defineComponent({ template: '<main><slot /></main>' })

function page(username: string, value = 10): LeaderboardData {
  return {
    items: [{ rank: 1, username, value, extra_int: 3 }],
    total: 1,
    page: 1,
    page_size: 20,
    pages: 1,
  }
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { promise, resolve }
}

function mountView() {
  return mount(LeaderboardView, {
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false, messages: { en: {} } })],
      stubs: {
        AppLayout: AppLayoutStub,
        Icon: true,
        LoadingSpinner: true,
        Pagination: true,
      },
    },
  })
}

describe('LeaderboardView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = {
      leaderboard_balance_enabled: true,
      leaderboard_consumption_enabled: true,
      leaderboard_checkin_enabled: true,
      leaderboard_transfer_enabled: true,
      transfer_enabled: true,
    }
    api.balance.mockResolvedValue(page('Balance user'))
    api.consumption.mockResolvedValue(page('Consumption user'))
    api.checkin.mockResolvedValue(page('Check-in user'))
    api.transfer.mockResolvedValue(page('Transfer user'))
  })

  it('loads the balance board on mount', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(api.balance).toHaveBeenCalledWith(1, 20)
    expect(wrapper.text()).toContain('Balance user')
  })

  it('requests the selected period for the consumption board', async () => {
    const wrapper = mountView()
    await flushPromises()
    const tabs = wrapper.findAll('button[role="tab"]')
    expect(tabs).toHaveLength(4)
    await tabs[1].trigger('click')
    await flushPromises()

    const periodButtons = wrapper.findAll('section[aria-label] button:not([role="tab"])')
    expect(periodButtons).toHaveLength(3)
    await periodButtons[2].trigger('click')
    await flushPromises()

    expect(api.consumption).toHaveBeenLastCalledWith('monthly', 1, 20)
    expect(wrapper.text()).toContain('Consumption user')
  })

  it('shows the closed state and skips requests when every board is disabled', async () => {
    appStore.cachedPublicSettings = {
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: false,
      leaderboard_checkin_enabled: false,
      leaderboard_transfer_enabled: false,
      transfer_enabled: false,
    }

    const wrapper = mountView()
    await flushPromises()

    expect(api.balance).not.toHaveBeenCalled()
    expect(api.consumption).not.toHaveBeenCalled()
    expect(wrapper.text()).toContain('leaderboard.noBoards')
  })

  it('does not let an older response overwrite a newly selected board', async () => {
    const balance = deferred<LeaderboardData>()
    const consumption = deferred<LeaderboardData>()
    api.balance.mockReturnValueOnce(balance.promise)
    api.consumption.mockReturnValueOnce(consumption.promise)

    const wrapper = mountView()
    await flushPromises()
    const tabs = wrapper.findAll('button[role="tab"]')
    await tabs[1].trigger('click')
    await flushPromises()

    consumption.resolve(page('Newest result'))
    await flushPromises()
    balance.resolve(page('Stale result'))
    await flushPromises()

    expect(wrapper.text()).toContain('Newest result')
    expect(wrapper.text()).not.toContain('Stale result')
  })
})
