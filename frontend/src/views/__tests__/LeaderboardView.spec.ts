import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'

import LeaderboardView from '../LeaderboardView.vue'
import { useAppStore } from '@/stores'

const {
  getBalanceLeaderboard,
  getConsumptionLeaderboard,
  getCheckinLeaderboard,
} = vi.hoisted(() => ({
  getBalanceLeaderboard: vi.fn(),
  getConsumptionLeaderboard: vi.fn(),
  getCheckinLeaderboard: vi.fn(),
}))

vi.mock('@/api/leaderboard', () => ({
  leaderboardAPI: {
    getBalanceLeaderboard,
    getConsumptionLeaderboard,
    getCheckinLeaderboard,
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const messages: Record<string, string> = {
    'common.loading': 'Loading',
    'leaderboard.title': 'Leaderboard',
    'leaderboard.subtitle': 'Top users',
    'leaderboard.tabs.balance': 'Balance',
    'leaderboard.tabs.consumption': 'Consumption',
    'leaderboard.tabs.transfer': 'Transfer',
    'leaderboard.tabs.checkin': 'Check-in',
    'leaderboard.tabsDisabled': 'Tabs disabled',
    'leaderboard.empty': 'No data',
    'leaderboard.periods.daily': 'Day',
    'leaderboard.periods.weekly': 'Week',
    'leaderboard.periods.monthly': 'Month',
    'leaderboard.balanceChartTitle': 'Balance Distribution',
    'leaderboard.balanceChartSubtitle': 'Balance subtitle',
    'leaderboard.consumptionChartTitle': 'Consumption Distribution',
    'leaderboard.consumptionChartSubtitle': 'Consumption subtitle',
    'leaderboard.checkinChartTitle': 'Check-in Distribution',
    'leaderboard.checkinChartSubtitle': 'Check-in subtitle',
    'leaderboard.amount': 'Amount',
    'leaderboard.requests': 'Requests',
    'leaderboard.checkins': 'Check-ins',
    'leaderboard.totalCheckins': 'Total Check-ins',
    'leaderboard.streak': 'Streak',
    'leaderboard.hoverHint': 'Hover amount',
    'leaderboard.checkinHoverHint': 'Hover streak',
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, fallback?: string) => messages[key] ?? fallback ?? key,
    }),
  }
})

function emptyLeaderboard() {
  return {
    items: [],
    total: 0,
    page: 1,
    page_size: 20,
    pages: 0,
  }
}

function mountView(settings: Record<string, boolean>) {
  const store = useAppStore()
  store.publicSettingsLoaded = true
  store.cachedPublicSettings = {
    site_name: 'Sub2API',
    site_logo: '',
    doc_url: '',
    ...settings,
  } as any

  return mount(LeaderboardView, {
    global: {
      stubs: {
        AppLayout: {
          template: '<div class="app-layout-stub"><h1>Leaderboard</h1><p>Top users</p><slot /></div>',
        },
        PublicLeaderboardChart: {
          props: ['chartItems', 'summary', 'entries', 'title', 'subtitle', 'valueLabel', 'metricLabel', 'hoverHint', 'valueType', 'subtitleType'],
          template: '<div class="leaderboard-chart-stub">{{ subtitleType }}|{{ chartItems.length }}|{{ summary?.total_value }}|{{ entries.length }}|{{ title }}</div>',
        },
      },
    },
  })
}

async function settleLeaderboard() {
  await flushPromises()
  await nextTick()
  await flushPromises()
  await nextTick()
}

describe('LeaderboardView tab switches', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    getBalanceLeaderboard.mockReset().mockResolvedValue(emptyLeaderboard())
    getConsumptionLeaderboard.mockReset().mockResolvedValue(emptyLeaderboard())
    getCheckinLeaderboard.mockReset().mockResolvedValue(emptyLeaderboard())
  })

  it('uses the layout heading without rendering a duplicate page title', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: true,
      leaderboard_consumption_enabled: false,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(wrapper.findAll('h1').map((heading) => heading.text())).toEqual(['Leaderboard'])
    expect((wrapper.text().match(/Top users/g) ?? []).length).toBe(1)
    expect(wrapper.get('[data-testid="leaderboard-page-shell"]').classes()).toContain('leaderboard-page-shell')
    expect(wrapper.findAll('.leaderboard-page-shell__glow')).toHaveLength(2)
  })

  it('shows only enabled public leaderboard tabs', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: true,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: true,
    })

    await settleLeaderboard()

    expect(wrapper.text()).not.toContain('Balance')
    expect(wrapper.text()).toContain('Consumption')
    expect(wrapper.text()).not.toContain('Transfer')
    expect(wrapper.text()).toContain('Check-in')
  })

  it('uses a clearly distinguishable active style for the selected leaderboard tab', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: true,
      leaderboard_consumption_enabled: true,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    const buttons = wrapper.findAll('button')
    const balanceButton = buttons.find((button) => button.text() === 'Balance')
    const consumptionButton = buttons.find((button) => button.text() === 'Consumption')

    expect(balanceButton).toBeTruthy()
    expect(consumptionButton).toBeTruthy()
    expect(balanceButton!.classes()).toContain('rounded-full')
    expect(balanceButton!.classes()).toContain('bg-primary-600')
    expect(balanceButton!.classes()).toContain('text-white')
    expect(consumptionButton!.classes()).toContain('border-transparent')

    await consumptionButton!.trigger('click')

    expect(consumptionButton!.classes()).toContain('rounded-full')
    expect(consumptionButton!.classes()).toContain('bg-primary-600')
    expect(consumptionButton!.classes()).toContain('text-white')
  })

  it('does not show transfer tab or call transfer endpoint', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: false,
      leaderboard_transfer_enabled: true,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(wrapper.text()).toContain('Tabs disabled')
    expect(wrapper.text()).not.toContain('Transfer')
    expect(getBalanceLeaderboard).not.toHaveBeenCalled()
    expect(getConsumptionLeaderboard).not.toHaveBeenCalled()
    expect(getCheckinLeaderboard).not.toHaveBeenCalled()
  })

  it('does not request leaderboard data when all public tabs are disabled', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: false,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(wrapper.text()).toContain('Tabs disabled')
    expect(getBalanceLeaderboard).not.toHaveBeenCalled()
    expect(getConsumptionLeaderboard).not.toHaveBeenCalled()
    expect(getCheckinLeaderboard).not.toHaveBeenCalled()
  })

  it('uses chart component for balance leaderboard', async () => {
    getBalanceLeaderboard.mockResolvedValueOnce({
      items: [
        { rank: 1, username: 'Alpha', value: 70, extra_int: 5 },
        { rank: 2, username: 'Beta', value: 30, extra_int: 4 },
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mountView({
      leaderboard_balance_enabled: true,
      leaderboard_consumption_enabled: false,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(wrapper.find('.leaderboard-chart-stub').text()).toBe('balance|2|100|2|Balance Distribution')
  })

  it('passes full chart_items to consumption chart', async () => {
    getConsumptionLeaderboard.mockResolvedValueOnce({
      items: [
        { rank: 1, username: 'Alpha', value: 70, extra_int: 5 },
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1,
      summary: {
        total_value: 100,
        total_users: 2,
      },
      chart_items: [
        { username: 'Alpha', value: 70 },
        { username: 'Beta', value: 30 },
      ],
    })

    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: true,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(wrapper.find('.leaderboard-chart-stub').text()).toBe('consumption|2|100|1|Consumption Distribution')
  })

  it('refetches all consumption items when total exceeds default page size', async () => {
    getConsumptionLeaderboard
      .mockResolvedValueOnce({
        items: Array.from({ length: 20 }, (_, index) => ({
          rank: index + 1,
          username: `User${index + 1}`,
          value: 100 - index,
          extra_int: index + 1,
        })),
        total: 23,
        page: 1,
        page_size: 20,
        pages: 2,
        summary: {
          total_value: 507.62,
          total_users: 23,
        },
        chart_items: Array.from({ length: 23 }, (_, index) => ({
          username: `User${index + 1}`,
          value: 100 - index,
        })),
      })
      .mockResolvedValueOnce({
        items: Array.from({ length: 23 }, (_, index) => ({
          rank: index + 1,
          username: `User${index + 1}`,
          value: 100 - index,
          extra_int: index + 1,
        })),
        total: 23,
        page: 1,
        page_size: 23,
        pages: 1,
        summary: {
          total_value: 507.62,
          total_users: 23,
        },
        chart_items: Array.from({ length: 23 }, (_, index) => ({
          username: `User${index + 1}`,
          value: 100 - index,
        })),
      })

    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: true,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(getConsumptionLeaderboard).toHaveBeenNthCalledWith(1, 'daily', 1, 20)
    expect(getConsumptionLeaderboard).toHaveBeenNthCalledWith(2, 'daily', 1, 23)
    expect(wrapper.find('.leaderboard-chart-stub').text()).toBe('consumption|23|507.62|23|Consumption Distribution')
  })
})
