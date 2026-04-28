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
  getTransferLeaderboard,
} = vi.hoisted(() => ({
  getBalanceLeaderboard: vi.fn(),
  getConsumptionLeaderboard: vi.fn(),
  getCheckinLeaderboard: vi.fn(),
  getTransferLeaderboard: vi.fn(),
}))

vi.mock('@/api/leaderboard', () => ({
  leaderboardAPI: {
    getBalanceLeaderboard,
    getConsumptionLeaderboard,
    getCheckinLeaderboard,
    getTransferLeaderboard,
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
    getTransferLeaderboard.mockReset().mockResolvedValue(emptyLeaderboard())
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

  it('四个标签全关时不请求排行榜接口', async () => {
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
    expect(getTransferLeaderboard).not.toHaveBeenCalled()
    expect(getCheckinLeaderboard).not.toHaveBeenCalled()
  })

  it('转账排行成为第一个可见标签时会使用转账周期', async () => {
    const wrapper = mountView({
      leaderboard_balance_enabled: false,
      leaderboard_consumption_enabled: false,
      leaderboard_transfer_enabled: true,
      leaderboard_checkin_enabled: false,
    })

    await settleLeaderboard()

    expect(wrapper.text()).toContain('转账排行')
    expect(getTransferLeaderboard).toHaveBeenCalledWith('day', 1, 20)
    expect(getConsumptionLeaderboard).not.toHaveBeenCalled()
  })
})
