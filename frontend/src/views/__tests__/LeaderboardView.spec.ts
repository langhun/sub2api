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
    'common.loading': '加载中',
    'leaderboard.title': '排行榜',
    'leaderboard.subtitle': '看看谁是活跃之星',
    'leaderboard.tabs.balance': '余额排行',
    'leaderboard.tabs.consumption': '消耗排行',
    'leaderboard.tabs.transfer': '转账排行',
    'leaderboard.tabs.checkin': '签到排行',
    'leaderboard.tabsDisabled': '排行榜标签已关闭',
    'leaderboard.empty': '暂无数据',
    'leaderboard.periods.daily': '日',
    'leaderboard.periods.weekly': '周',
    'leaderboard.periods.monthly': '月',
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
        PublicPageHeader: { template: '<header />' },
        PublicPageFooter: { template: '<footer />' },
        PublicConsumptionLeaderboardChart: {
          props: ['chartItems', 'summary', 'entries'],
          template: '<div class="consumption-chart-stub">{{ chartItems.length }}|{{ summary?.total_value }}|{{ entries.length }}</div>',
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

  it('按独立开关只显示启用的排行榜标签', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: true,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: true,
    })

    await settleLeaderboard()

    expect(wrapper.text()).not.toContain('余额排行')
    expect(wrapper.text()).toContain('消耗排行')
    expect(wrapper.text()).not.toContain('转账排行')
    expect(wrapper.text()).toContain('签到排行')
    expect(getBalanceLeaderboard).not.toHaveBeenCalled()
    expect(getConsumptionLeaderboard).toHaveBeenCalledWith('daily', 1, 20)
  })

  it('公开排行榜即使开启 transfer 开关也不会显示转账标签或请求 transfer 接口', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: false,
      leaderboard_transfer_enabled: true,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(wrapper.text()).toContain('排行榜标签已关闭')
    expect(wrapper.text()).not.toContain('转账排行')
    expect(getBalanceLeaderboard).not.toHaveBeenCalled()
    expect(getConsumptionLeaderboard).not.toHaveBeenCalled()
    expect(getCheckinLeaderboard).not.toHaveBeenCalled()
  })

  it('三个公开标签全关时不请求排行榜接口', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: false,
      leaderboard_transfer_enabled: false,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(wrapper.text()).toContain('排行榜标签已关闭')
    expect(getBalanceLeaderboard).not.toHaveBeenCalled()
    expect(getConsumptionLeaderboard).not.toHaveBeenCalled()
    expect(getCheckinLeaderboard).not.toHaveBeenCalled()
  })

  it('消费榜图表使用全量 chart_items，列表显示金额占比', async () => {
    getConsumptionLeaderboard.mockResolvedValueOnce({
      items: [
        {
          rank: 1,
          username: 'Alpha',
          value: 70,
          extra_int: 5,
        },
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

    expect(wrapper.find('.consumption-chart-stub').text()).toBe('2|100|1')
  })

  it('消费榜用户数超过默认 20 条时会补拉全量榜单条目', async () => {
    getConsumptionLeaderboard
      .mockResolvedValueOnce({
        items: Array.from({ length: 20 }, (_, index) => ({
          rank: index + 1,
          username: `用户${index + 1}`,
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
          username: `用户${index + 1}`,
          value: 100 - index,
        })),
      })
      .mockResolvedValueOnce({
        items: Array.from({ length: 23 }, (_, index) => ({
          rank: index + 1,
          username: `用户${index + 1}`,
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
          username: `用户${index + 1}`,
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
    expect(wrapper.find('.consumption-chart-stub').text()).toBe('23|507.62|23')
  })
})
