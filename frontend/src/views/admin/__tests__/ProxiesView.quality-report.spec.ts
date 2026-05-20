import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ProxiesView from '../ProxiesView.vue'

const {
  listProxies,
  checkProxyQuality,
  getAllGroups,
  listProxySubscriptions,
  showError,
  showSuccess,
  showInfo
} = vi.hoisted(() => ({
  listProxies: vi.fn(),
  checkProxyQuality: vi.fn(),
  getAllGroups: vi.fn(),
  listProxySubscriptions: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showInfo: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      list: listProxies,
      create: vi.fn(),
      batchCreate: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      batchDelete: vi.fn(),
      unassignAccounts: vi.fn(),
      getProxyAccounts: vi.fn(),
      testProxy: vi.fn(),
      checkProxyQuality,
      exportData: vi.fn(),
      importData: vi.fn(),
      updatePoolMembership: vi.fn(),
      clearCooldown: vi.fn()
    },
    proxySubscriptions: {
      list: listProxySubscriptions,
      refresh: vi.fn(),
      listNodes: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      delete: vi.fn()
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showInfo
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const translations: Record<string, string> = {
    'admin.proxies.qualityCheck': '质量检测',
    'admin.proxies.qualityCheckDone': '质量检测完成',
    'admin.proxies.qualityReportTitle': '代理质量检测报告',
    'admin.proxies.qualityScoreLabel': '评分',
    'admin.proxies.qualityGradeLabel': '等级',
    'admin.proxies.qualityBaseLatency': '基础延迟',
    'admin.proxies.qualityCountry': '出口地区',
    'admin.proxies.qualityExitIP': '出口 IP',
    'admin.proxies.qualityCheckedAt': '检测时间',
    'admin.proxies.qualityInterpretation': '结果解读',
    'admin.proxies.qualityInterpretationWarn': '目标可达，但部分平台要求鉴权或方法受限。',
    'admin.proxies.qualityStatusPass': '通过',
    'admin.proxies.qualityStatusWarn': '告警',
    'admin.proxies.qualityStatusChallenge': '挑战',
    'admin.proxies.qualityStatusFail': '失败',
    'admin.proxies.qualityStatusHealthy': '优质',
    'admin.proxies.qualityTableTarget': '检测项',
    'admin.proxies.qualityTableStatus': '状态',
    'admin.proxies.qualityTableLatency': '延迟',
    'admin.proxies.qualityTableMessage': '说明',
    'admin.proxies.qualityTargetBase': '基础连通性',
    'common.close': '关闭',
    'common.edit': '编辑',
    'common.more': '更多',
    'admin.accounts.status.active': '启用',
    'admin.accounts.status.inactive': '停用'
  }

  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (key === 'admin.proxies.qualityInline') {
          return `质量 ${params?.grade}/${params?.score}`
        }
        return translations[key] ?? key
      }
    })
  }
})

vi.mock('@/composables/useTableSelection', () => ({
  useTableSelection: () => ({
    selectedSet: new Set<number>(),
    selectedCount: { value: 0 },
    allVisibleSelected: false,
    isSelected: () => false,
    select: vi.fn(),
    deselect: vi.fn(),
    clear: vi.fn(),
    removeMany: vi.fn(),
    toggleVisible: vi.fn(),
    batchUpdate: vi.fn()
  })
}))

vi.mock('@/composables/useSwipeSelect', () => ({
  useSwipeSelect: vi.fn()
}))

const DataTableStub = {
  props: ['data', 'columns'],
  template: `
    <div data-test="data-table-stub">
      <div v-for="row in data" :key="row.id" data-test="proxy-row">
        <slot name="cell-actions" :row="row" :value="row.actions" />
      </div>
    </div>
  `
}

const BaseDialogStub = {
  props: ['show', 'title', 'width'],
  template: `
    <div v-if="show" data-test="base-dialog-stub" :data-title="title" :data-width="width">
      <slot />
      <slot name="footer" />
    </div>
  `
}

const ProxiesToolbarStub = {
  props: [
    'activeTab',
    'searchQuery',
    'filters',
    'protocolOptions',
    'statusOptions',
    'runtimeStatusOptions',
    'loading',
    'loadingSubscriptions',
    'batchTesting',
    'batchQualityChecking',
    'selectedCount',
    'showColumnDropdown',
    'showProxyToolsDropdown',
    'showProxyBatchDropdown',
    'toggleableColumns',
    'isColumnVisible'
  ],
  template: '<div data-test="proxies-toolbar-stub"></div>'
}

const mountView = () =>
  mount(ProxiesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        ProxiesToolbar: ProxiesToolbarStub,
        ProxySubscriptionsPanel: true,
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        EmptyState: true,
        ImportDataModal: true,
        AssignAccountsModal: true,
        PoolMembersDialog: true,
        Select: true,
        Icon: true,
        Teleport: true
      }
    }
  })

describe('admin ProxiesView quality report', () => {
  beforeEach(() => {
    localStorage.clear()
    listProxies.mockReset()
    checkProxyQuality.mockReset()
    getAllGroups.mockReset()
    listProxySubscriptions.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    showInfo.mockReset()

    listProxies.mockResolvedValue({
      items: [
        {
          id: 6,
          name: 'sub 自动选路入口 #6',
          protocol: 'http',
          host: 'proxy.example.com',
          port: 8080,
          username: null,
          password: null,
          status: 'active',
          health_status: 'healthy',
          auto_failover_pool_enabled: false,
          created_at: '2026-05-20T00:00:00Z',
          updated_at: '2026-05-20T00:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    checkProxyQuality.mockResolvedValue({
      proxy_id: 6,
      score: 80,
      grade: 'B',
      summary: '通过 2 项，告警 2 项，失败 0 项，挑战 0 项',
      exit_ip: '13.112.14.79',
      country: 'JP',
      country_code: 'jp',
      base_latency_ms: 125,
      passed_count: 2,
      warn_count: 2,
      failed_count: 0,
      challenge_count: 0,
      checked_at: 1770000000,
      items: [
        {
          target: 'base_connectivity',
          status: 'pass',
          latency_ms: 125,
          message: '代理出口连通正常'
        },
        {
          target: 'openai',
          status: 'warn',
          http_status: 401,
          latency_ms: 263,
          message: 'HTTP 401（目标可达，但鉴权或方法受限）'
        },
        {
          target: 'anthropic',
          status: 'warn',
          http_status: 405,
          latency_ms: 118,
          message: 'HTTP 405（目标可达，但鉴权或方法受限）'
        },
        {
          target: 'gemini',
          status: 'pass',
          http_status: 200,
          latency_ms: 181,
          message: 'HTTP 200'
        }
      ]
    })

    getAllGroups.mockResolvedValue([])
    listProxySubscriptions.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 100,
      pages: 0
    })
  })

  it('renders the quality report in a wide dialog with clearer warn messaging', async () => {
    const wrapper = mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button[aria-label="质量检测"]')
    expect(buttons).toHaveLength(1)

    await buttons[0].trigger('click')
    await flushPromises()

    expect(checkProxyQuality).toHaveBeenCalledWith(6)

    const dialog = wrapper.get('[data-test="base-dialog-stub"]')
    expect(dialog.attributes('data-width')).toBe('wide')
    expect(dialog.attributes('data-title')).toBe('代理质量检测报告')
    expect(dialog.text()).toContain('评分')
    expect(dialog.text()).toContain('80')
    expect(dialog.text()).toContain('等级')
    expect(dialog.text()).toContain('B')
    expect(dialog.text()).toContain('结果解读')
    expect(dialog.text()).toContain('目标可达，但部分平台要求鉴权或方法受限。')
    expect(dialog.text()).toContain('HTTP 401（目标可达，但鉴权或方法受限）')
    expect(dialog.text()).toContain('HTTP 405（目标可达，但鉴权或方法受限）')
    expect(dialog.text()).toContain('基础连通性')
    expect(dialog.text()).toContain('OpenAI')
    expect(dialog.text()).toContain('Anthropic')
    expect(dialog.text()).toContain('Gemini')

    const badges = wrapper.findAll('.badge-purple')
    expect(badges).toHaveLength(0)
  })
})
