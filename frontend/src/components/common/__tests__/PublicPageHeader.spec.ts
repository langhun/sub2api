import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import PublicPageHeader from '../PublicPageHeader.vue'

const storeMocks = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {
      site_name: 'Sub2API',
      site_logo: '',
      doc_url: '/docs',
      home_nav_links_enabled: true,
      home_nav_leaderboard_enabled: true,
      home_nav_key_usage_enabled: true,
      home_nav_monitoring_enabled: true,
      home_nav_pricing_enabled: true,
      leaderboard_balance_enabled: true,
      leaderboard_consumption_enabled: true,
      leaderboard_transfer_enabled: true,
      leaderboard_checkin_enabled: true,
    },
    siteName: 'Sub2API',
    siteLogo: '',
    docUrl: '/docs',
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    checkAuth: vi.fn(),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => storeMocks.appStore,
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => storeMocks.authStore,
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => ({
        'leaderboard.title': '排行榜',
        'home.keyUsage': '用量查询',
        'admin.monitoring.title': '平台监控',
        'pricing.title': '模型定价',
        'home.docs': '文档',
        'home.login': '登录',
        'home.dashboard': '控制台',
      })[key] || key,
    }),
  }
})

function mountHeader(props: Record<string, unknown> = {}) {
  return mount(PublicPageHeader, {
    props,
    global: {
      stubs: {
        RouterLink: {
          props: ['to'],
          template: '<a><slot /></a>',
        },
        LocaleSwitcher: true,
        Icon: true,
      },
    },
  })
}

describe('PublicPageHeader', () => {
  beforeEach(() => {
    localStorage.setItem('theme', 'light')
    document.documentElement.classList.remove('dark')
    storeMocks.appStore.fetchPublicSettings.mockClear()
    storeMocks.authStore.checkAuth.mockClear()
  })

  it('默认显示首页顶部入口', () => {
    const wrapper = mountHeader()

    expect(wrapper.text()).toContain('排行榜')
    expect(wrapper.text()).toContain('用量查询')
    expect(wrapper.text()).toContain('平台监控')
    expect(wrapper.text()).toContain('模型定价')
  })

  it.each([
    [{ leaderboard: false }, '排行榜', ['用量查询', '平台监控', '模型定价']],
    [{ keyUsage: false }, '用量查询', ['排行榜', '平台监控', '模型定价']],
    [{ monitoring: false }, '平台监控', ['排行榜', '用量查询', '模型定价']],
    [{ pricing: false }, '模型定价', ['排行榜', '用量查询', '平台监控']],
  ])('关闭单个入口时只隐藏对应入口 %#', (navLinkVisibility, hiddenLabel, visibleLabels) => {
    const wrapper = mountHeader({ navLinkVisibility })

    expect(wrapper.text()).not.toContain(hiddenLabel)
    for (const label of visibleLabels) {
      expect(wrapper.text()).toContain(label)
    }
    expect(wrapper.text()).toContain('文档')
  })

  it('全部关闭时隐藏四个首页顶部入口并保留文档入口', () => {
    const wrapper = mountHeader({
      navLinkVisibility: {
        leaderboard: false,
        keyUsage: false,
        monitoring: false,
        pricing: false,
      },
    })

    expect(wrapper.text()).not.toContain('排行榜')
    expect(wrapper.text()).not.toContain('用量查询')
    expect(wrapper.text()).not.toContain('平台监控')
    expect(wrapper.text()).not.toContain('模型定价')
    expect(wrapper.text()).toContain('文档')
  })
})
