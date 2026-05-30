import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ProxiesView from '../ProxiesView.vue'
import PoolMembersDialog from '@/components/admin/proxy/PoolMembersDialog.vue'

const {
  listProxies,
  updateProxy,
  updatePoolMembership,
  clearCooldown,
  getAllGroups,
  listProxySubscriptions,
  refreshProxySubscription,
  listSubscriptionNodes,
  createProxySubscription,
  showError,
  showSuccess,
  showInfo,
  showWarning
} = vi.hoisted(() => ({
  listProxies: vi.fn(),
  updateProxy: vi.fn(),
  updatePoolMembership: vi.fn(),
  clearCooldown: vi.fn(),
  getAllGroups: vi.fn(),
  listProxySubscriptions: vi.fn(),
  refreshProxySubscription: vi.fn(),
  listSubscriptionNodes: vi.fn(),
  createProxySubscription: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showInfo: vi.fn(),
  showWarning: vi.fn()
}))

const keyboardShortcutBindings = vi.hoisted(() => ({
  current: null as null | { onRefresh?: () => void }
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      list: listProxies,
      update: updateProxy,
      updatePoolMembership,
      clearCooldown
    },
    proxySubscriptions: {
      list: listProxySubscriptions,
      refresh: refreshProxySubscription,
      listNodes: listSubscriptionNodes,
      create: createProxySubscription,
      update: vi.fn(),
      delete: vi.fn()
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/composables/useKeyboardShortcuts', () => ({
  useKeyboardShortcuts: (config: { onRefresh?: () => void }) => {
    keyboardShortcutBindings.current = config
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showInfo,
    showWarning
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const translations: Record<string, string> = {
    'admin.proxies.disableAction': '停用代理',
    'admin.proxies.enableAction': '启用代理',
    'admin.proxies.statusDisabled': '代理已停用',
    'admin.proxies.statusEnabled': '代理已启用'
  }
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => translations[key] ?? key
    })
  }
})

const DataTableStub = {
  props: ['data', 'columns'],
  template: `
    <div data-test="data-table-stub" :data-columns="columns.map(column => column.key).join(',')">
      <slot name="header-select" />
      <div v-for="row in data" :key="row.id" data-test="proxy-row" :data-row-id="row.id">
        <slot name="cell-select" :row="row" :value="row.id" />
        <slot name="cell-location" :row="row" :value="row.location" />
        <slot name="cell-status" :row="row" :value="row.status" />
        <slot name="cell-actions" :row="row" :value="row.actions" />
      </div>
    </div>
  `
}

const SelectStub = {
  props: ['modelValue', 'options', 'placeholder', 'disabled'],
  template: '<div data-test="select-stub" :data-placeholder="placeholder" :data-disabled="disabled ? \'true\' : \'false\'"></div>'
}

const BaseDialogStub = {
  props: ['show', 'title', 'width'],
  template: `
    <div v-if="show" data-test="base-dialog-stub">
      <slot />
      <slot name="footer" />
    </div>
  `
}

const ProxyBulkActionsBarStub = {
  props: ['selectedCount'],
  emits: ['test', 'quality-check', 'enable-pool', 'disable-pool', 'clear-cooldown', 'assign', 'unassign', 'delete', 'clear'],
  template: `
    <div v-if="selectedCount > 0" data-test="proxy-bulk-bar-stub">
      <button data-test="bulk-disable-pool" @click="$emit('disable-pool')">admin.proxies.poolBatchDisable</button>
    </div>
  `
}

const mountProxiesView = () =>
  mount(ProxiesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        EmptyState: true,
        ImportDataModal: true,
        AssignAccountsModal: true,
        ProxyBulkActionsBar: ProxyBulkActionsBarStub,
        PoolMembersDialog: true,
        Select: SelectStub,
        Icon: true
      }
    }
  })

describe('admin ProxiesView pool state', () => {
  beforeEach(() => {
    localStorage.clear()
    listProxies.mockReset()
    updateProxy.mockReset()
    updatePoolMembership.mockReset()
    clearCooldown.mockReset()
    getAllGroups.mockReset()
    listProxySubscriptions.mockReset()
    refreshProxySubscription.mockReset()
    listSubscriptionNodes.mockReset()
    createProxySubscription.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    showInfo.mockReset()
    showWarning.mockReset()
    keyboardShortcutBindings.current = null

    listProxies.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'pool-a',
          protocol: 'http',
          host: 'proxy.example.com',
          port: 8080,
          username: null,
          password: null,
          status: 'active',
          country: '美国',
          country_code: 'us',
          region: '加利福尼亚',
          city: '洛杉矶',
          auto_failover_pool_enabled: true,
          health_status: 'cooldown',
          cooldown_until_unix: Math.floor(Date.now() / 1000) + 120,
          failover_switch_count: 3,
          created_at: '2026-05-07T00:00:00Z',
          updated_at: '2026-05-07T00:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getAllGroups.mockResolvedValue([])
    listProxySubscriptions.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 100,
      pages: 0
    })
    refreshProxySubscription.mockResolvedValue({
      source_id: 1,
      refreshed_at: 1710000000,
      node_count: 3,
      materialized_proxy_count: 2,
      created_proxy_count: 0,
      updated_proxy_count: 0,
      disabled_proxy_count: 0,
      deleted_proxy_count: 0,
      skipped_node_count: 0,
      conflict_node_count: 0,
      unsupported_node_count: 0,
      errors: []
    })
    listSubscriptionNodes.mockResolvedValue([
      {
        id: 9,
        source_id: 1,
        node_key: 'node-1',
        display_name: 'Node One',
        node_type: 'http',
        server: '1.1.1.1',
        port: 8080,
        config_json: {},
        landing_status: 'active',
        last_error: '',
        last_seen_at: '2026-05-19T10:00:00Z',
        created_at: '2026-05-19T10:00:00Z',
        updated_at: '2026-05-19T10:00:00Z'
      }
    ])
    createProxySubscription.mockResolvedValue({
      id: 2,
      name: 'sub-new',
      url: 'https://example.com/new',
      source_format: 'auto',
      enabled: true,
      refresh_interval_hours: 6,
      target_entry_count: 1,
      auto_add_to_pool: true,
      last_refreshed_at: null,
      last_success_at: null,
      last_error: '',
      last_node_count: 0,
      last_materialized_proxy_count: 0,
      created_at: '2026-05-19T10:00:00Z',
      updated_at: '2026-05-19T10:00:00Z'
    })
    updateProxy.mockResolvedValue(undefined)
    updatePoolMembership.mockResolvedValue(undefined)
    clearCooldown.mockResolvedValue(undefined)
  })

  it('opens the pool members dialog from the tools menu', async () => {
    const wrapper = mount(ProxiesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          ImportDataModal: true,
          AssignAccountsModal: true,
          PoolMembersDialog,
          Select: SelectStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    const openPoolButton = wrapper.get('[data-test="proxy-toolbar-pool"]')
    await openPoolButton.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.proxies.poolMembersSummary')
    expect(wrapper.text()).toContain('admin.proxies.poolUsageHint')
    expect(wrapper.text()).toContain('pool-a')
    expect(wrapper.text()).toContain('admin.proxies.healthCooldown')
  })

  it('renders location with an inline flag instead of a remote image', async () => {
    const wrapper = mount(ProxiesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          ImportDataModal: true,
          AssignAccountsModal: true,
          PoolMembersDialog,
          Select: SelectStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('🇺🇸')
    expect(wrapper.text()).toContain('美国 · 加利福尼亚 · 洛杉矶')
    expect(wrapper.find('img').exists()).toBe(false)
  })

  it('supports proxy column settings and keeps row actions compact behind a more menu', async () => {
    const wrapper = mount(ProxiesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          ImportDataModal: true,
          AssignAccountsModal: true,
          PoolMembersDialog: true,
          Select: SelectStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    const toolsButton = wrapper.get('[data-test="proxy-toolbar-tools"]')
    await toolsButton.trigger('click')
    await flushPromises()

    const columnSettingsButton = wrapper.get('[data-test="proxy-toolbar-columns"]')

    expect(columnSettingsButton).toBeDefined()
    await columnSettingsButton!.trigger('click')
    await flushPromises()

    const authToggleButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.proxies.columns.auth'))

    expect(authToggleButton).toBeDefined()
    await authToggleButton!.trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="data-table-stub"]').attributes('data-columns')).not.toContain('auth')

    const moreButton = wrapper
      .findAll('button')
      .find((button) => button.attributes('title') === 'common.more')

    expect(moreButton).toBeDefined()
  })

  it('separates config status and runtime status filters', async () => {
    const wrapper = mount(ProxiesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          ImportDataModal: true,
          AssignAccountsModal: true,
          PoolMembersDialog: true,
          Select: SelectStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    const selectStubs = wrapper.findAll('[data-test="select-stub"]')
    const placeholders = selectStubs.map((node) => node.attributes('data-placeholder'))
    expect(placeholders).toContain('admin.proxies.allStatus')
    expect(placeholders).toContain('admin.proxies.allRuntimeStatus')
  })

  it('opens the Mihomo dialog and loads shared subscription sources', async () => {
    const wrapper = mount(ProxiesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: { template: '<div data-test="pagination-stub"></div>' },
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          ImportDataModal: true,
          AssignAccountsModal: true,
          PoolMembersDialog: true,
          Select: SelectStub,
          Icon: true
        }
      }
    })

    await flushPromises()
    const openMihomoButton = wrapper.get('[data-test="proxy-mihomo-config"]')
    await openMihomoButton.trigger('click')
    await flushPromises()

    expect(getMihomo).toHaveBeenCalledTimes(1)
    expect(listProxySubscriptions).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('admin.proxies.mihomo.workspaceDescription')
    expect(wrapper.text()).toContain('admin.proxies.mihomo.workspaceTitle')
    expect(wrapper.text()).toContain('admin.proxies.mihomo.sourceSectionTitle')
    expect(wrapper.text()).toContain('admin.proxies.mihomo.workspaceStatusEmpty')
    expect(wrapper.text()).toContain('admin.proxies.mihomo.listenerSummaryTitle')
  })

  it('uses backend Mihomo settings instead of the default form when subscription sync starts before Mihomo load finishes', async () => {
    const pendingMihomoLoad = new Promise(() => {})
    const loadedStatus = {
      settings: {
        protocol: 'http',
        target_host: 'mihomo-sub2api',
        start_port: 52001,
        listener_count: 6,
        controller_url: 'http://mihomo-sub2api:9097',
        controller_secret: 'secret-token',
        proxy_name_prefix: 'workspace',
        listener_regions: ['香港', '日本', '', '', '', ''],
        auto_optimize: false,
        country_filter: '日本'
      },
      config_path: '/tmp/mihomo/runtime.yaml',
      proxies: [],
      available_regions: ['香港', '日本']
    }
    getMihomo.mockReset()
    getMihomo
      .mockImplementationOnce(() => pendingMihomoLoad)
      .mockResolvedValue(loadedStatus)

    const wrapper = mountProxiesView()
    await flushPromises()

    const openMihomoButton = wrapper.get('[data-test="proxy-mihomo-config"]')
    await openMihomoButton.trigger('click')
    await flushPromises()

    const createButton = wrapper.findAll('button').find((button) => button.text().includes('admin.proxies.subscriptions.create'))
    expect(createButton).toBeDefined()
    await createButton!.trigger('click')
    await flushPromises()

    const textInputs = wrapper.findAll('input[type="text"], input[type="url"]')
    await textInputs[0].setValue('race-source')
    await textInputs[1].setValue('https://example.com/race')
    await wrapper.find('form#create-subscription-form').trigger('submit')
    await flushPromises()

    expect(createProxySubscription).toHaveBeenCalled()
    expect(syncMihomo).toHaveBeenCalledTimes(1)
    expect(syncMihomo.mock.calls[0]?.[0]).toEqual(expect.objectContaining({
      protocol: 'http',
      target_host: 'mihomo-sub2api',
      start_port: 52001,
      listener_count: 6,
      controller_url: 'http://mihomo-sub2api:9097',
      controller_secret: 'secret-token',
      proxy_name_prefix: 'workspace',
      country_filter: '日本'
    }))
    expect(getMihomo.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('auto-picks Mihomo strategy, country, and listener regions before sync', async () => {
    getMihomo.mockResolvedValueOnce({
      settings: {
        protocol: 'socks5h',
        target_host: '127.0.0.1',
        start_port: 41001,
        listener_count: 3,
        controller_url: 'http://127.0.0.1:9097',
        controller_secret: '',
        proxy_name_prefix: 'mihomo',
        listener_regions: ['', '', ''],
        auto_optimize: false,
        country_filter: ''
      },
      config_path: '/tmp/mihomo/config.yaml',
      proxies: [],
      available_regions: ['香港', '日本']
    })

    const wrapper = mountProxiesView()
    await flushPromises()

    await wrapper.get('[data-test="proxy-mihomo-config"]').trigger('click')
    await flushPromises()

    const autoCountryButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.proxies.mihomo.applyRecommendedCountry'))
    expect(autoCountryButton).toBeDefined()
    await autoCountryButton!.trigger('click')
    await flushPromises()

    const autoFillButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.proxies.mihomo.autoFillListenerRegions'))
    expect(autoFillButton).toBeDefined()
    await autoFillButton!.trigger('click')
    await flushPromises()

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.proxies.mihomo.saveAndSync'))!
      .trigger('click')
    await flushPromises()

    expect(syncMihomo).toHaveBeenCalledWith(expect.objectContaining({
      auto_optimize: false,
      country_filter: '香港',
      listener_regions: ['香港', '日本', '香港']
    }))

    const autoStrategyButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.proxies.mihomo.applyAutoOptimize'))
    expect(autoStrategyButton).toBeDefined()
    await autoStrategyButton!.trigger('click')
    await flushPromises()

    await wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.proxies.mihomo.saveAndSync'))!
      .trigger('click')
    await flushPromises()

    expect(syncMihomo).toHaveBeenLastCalledWith(expect.objectContaining({
      auto_optimize: true,
      country_filter: '',
      listener_regions: ['香港', '日本', '香港']
    }))
  })

  it('renders shared subscription sources in the Mihomo dialog and wires refresh, nodes, and create flows', async () => {
    listProxySubscriptions.mockResolvedValueOnce({
      items: [
        {
          id: 1,
          name: 'sub-a',
          url: 'https://example.com/sub',
          source_format: 'auto',
          enabled: true,
          refresh_interval_hours: 6,
          target_entry_count: 1,
          auto_add_to_pool: true,
          last_refreshed_at: '2026-05-19T10:00:00Z',
          last_success_at: null,
          last_error: '',
          last_node_count: 3,
          last_materialized_proxy_count: 2,
          created_at: '2026-05-19T10:00:00Z',
          updated_at: '2026-05-19T10:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 100,
      pages: 1
    })

    const wrapper = mount(ProxiesView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: { template: '<div data-test="pagination-stub"></div>' },
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          ImportDataModal: true,
          AssignAccountsModal: true,
          PoolMembersDialog: true,
          Select: SelectStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    const openMihomoButton = wrapper.get('[data-test="proxy-mihomo-config"]')
    await openMihomoButton.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('sub-a')
    expect(wrapper.text()).toContain('https://example.com/sub')

    const nodesButton = wrapper.findAll('button').find((button) => button.text().includes('admin.proxies.subscriptions.viewNodes'))
    expect(nodesButton).toBeDefined()
    await nodesButton!.trigger('click')
    await flushPromises()
    expect(listSubscriptionNodes).toHaveBeenCalledWith(1)
    expect(wrapper.text()).toContain('Node One')

    const refreshButton = wrapper.findAll('button').find((button) => button.text().includes('admin.proxies.subscriptions.refreshNow'))
    expect(refreshButton).toBeDefined()
    await refreshButton!.trigger('click')
    await flushPromises()
    expect(refreshProxySubscription).toHaveBeenCalledWith(1)
    expect(syncMihomo).toHaveBeenCalledTimes(1)

    const createButton = wrapper.findAll('button').find((button) => button.text().includes('admin.proxies.subscriptions.create'))
    expect(createButton).toBeDefined()
    await createButton!.trigger('click')
    await flushPromises()

    const textInputs = wrapper.findAll('input[type="text"], input[type="url"]')
    await textInputs[0].setValue('sub-new')
    await textInputs[1].setValue('https://example.com/new')
    await wrapper.find('form#create-subscription-form').trigger('submit')
    await flushPromises()
    expect(createProxySubscription).toHaveBeenCalled()
    expect(syncMihomo).toHaveBeenCalledTimes(2)
  })

  it('loads every subscription page when the Mihomo dialog is opened', async () => {
    listProxySubscriptions
      .mockResolvedValueOnce({
        items: [
          {
            id: 1,
            name: 'sub-a',
            url: 'https://example.com/sub-a',
            source_format: 'auto',
            enabled: true,
            refresh_interval_hours: 6,
            target_entry_count: 1,
            auto_add_to_pool: true,
            last_refreshed_at: null,
            last_success_at: null,
            last_error: '',
            last_node_count: 1,
            last_materialized_proxy_count: 1,
            created_at: '2026-05-19T10:00:00Z',
            updated_at: '2026-05-19T10:00:00Z'
          }
        ],
        total: 2,
        page: 1,
        page_size: 100,
        pages: 2
      })
      .mockResolvedValueOnce({
        items: [
          {
            id: 2,
            name: 'sub-b',
            url: 'https://example.com/sub-b',
            source_format: 'auto',
            enabled: true,
            refresh_interval_hours: 6,
            target_entry_count: 1,
            auto_add_to_pool: true,
            last_refreshed_at: null,
            last_success_at: null,
            last_error: '',
            last_node_count: 1,
            last_materialized_proxy_count: 1,
            created_at: '2026-05-19T10:00:00Z',
            updated_at: '2026-05-19T10:00:00Z'
          }
        ],
        total: 2,
        page: 2,
        page_size: 100,
        pages: 2
      })

    const wrapper = mountProxiesView()
    await flushPromises()

    const openMihomoButton = wrapper.get('[data-test="proxy-mihomo-config"]')
    await openMihomoButton.trigger('click')
    await flushPromises()

    expect(listProxySubscriptions).toHaveBeenNthCalledWith(1, 1, 100)
    expect(listProxySubscriptions).toHaveBeenNthCalledWith(2, 2, 100)
    expect(wrapper.text()).toContain('sub-a')
    expect(wrapper.text()).toContain('sub-b')
  })

  it('keeps the refresh shortcut bound to the proxy list outside the Mihomo dialog', async () => {
    mountProxiesView()
    await flushPromises()

    const proxyCallCountBeforeRefresh = listProxies.mock.calls.length
    expect(keyboardShortcutBindings.current?.onRefresh).toBeTypeOf('function')
    keyboardShortcutBindings.current?.onRefresh?.()
    await flushPromises()

    expect(listProxySubscriptions).not.toHaveBeenCalled()
    expect(listProxies.mock.calls.length).toBeGreaterThan(proxyCallCountBeforeRefresh)
  })

  it('shows the subscription-managed readonly hint and still submits editable fields', async () => {
    listProxies.mockResolvedValueOnce({
      items: [
        {
          id: 99,
          name: 'managed-proxy',
          protocol: 'socks5',
          host: 'managed.example.com',
          port: 1080,
          username: 'managed-user',
          password: 'managed-pass',
          status: 'active',
          country: '日本',
          country_code: 'jp',
          city: '东京',
          auto_failover_pool_enabled: true,
          managed_by_subscription: true,
          subscription_source_name: 'sub-source',
          subscription_node_type: 'vmess',
          created_at: '2026-05-18T00:00:00Z',
          updated_at: '2026-05-18T00:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })

    const wrapper = mountProxiesView()
    await flushPromises()

    const editButton = wrapper.findAll('button').find((button) => button.attributes('title') === 'common.edit')
    expect(editButton).toBeDefined()
    await editButton!.trigger('click')
    await flushPromises()

    const editForm = wrapper.get('form#edit-proxy-form')
    expect(wrapper.text()).toContain('admin.proxies.subscriptions.managedReadonlyHint')

    const textInputs = editForm.findAll('input[type="text"]')
    expect(textInputs).toHaveLength(3)
    expect(textInputs[0].attributes('disabled')).toBeDefined()
    expect(textInputs[1].attributes('disabled')).toBeDefined()
    expect(textInputs[2].attributes('disabled')).toBeDefined()
    expect(editForm.get('input[type="number"]').attributes('disabled')).toBeDefined()
    expect(editForm.get('input[type="password"]').attributes('disabled')).toBeDefined()

    const selects = editForm.findAll('[data-test="select-stub"]')
    expect(selects[0].attributes('data-disabled')).toBe('true')
    expect(selects[1].attributes('data-disabled')).toBe('false')

    await editForm.get('input[type="checkbox"]').setValue(false)
    await editForm.trigger('submit')
    await flushPromises()

    expect(updateProxy).toHaveBeenCalledWith(99, {
      name: 'managed-proxy',
      protocol: 'socks5',
      host: 'managed.example.com',
      port: 1080,
      username: 'managed-user',
      status: 'active',
      auto_failover_pool_enabled: false
    })
    expect(updateProxy.mock.calls[0]?.[1]).not.toHaveProperty('password')
  })

  it('surfaces batch actions from the sticky batch bar once rows are selected', async () => {
    const wrapper = mountProxiesView()
    await flushPromises()

    expect(wrapper.text()).not.toContain('admin.proxies.poolDisableAction')

    const row = wrapper.get('[data-test="proxy-row"][data-row-id="1"]')
    await row.get('input[type="checkbox"]').setValue(true)
    await flushPromises()

    const poolDisableButton = wrapper.get('[data-test="bulk-disable-pool"]')
    await poolDisableButton.trigger('click')
    await flushPromises()

    expect(updatePoolMembership).toHaveBeenCalledWith([1], false)
    expect(listProxies.mock.calls.length).toBeGreaterThanOrEqual(2)
    expect(wrapper.text()).not.toContain('admin.proxies.poolDisableAction')
    expect(wrapper.get('[data-test="proxy-row"][data-row-id="1"]').get('input[type="checkbox"]').element.checked).toBe(false)
  })

  it('restores the row-level proxy status toggle from the more menu', async () => {
    const wrapper = mountProxiesView()
    await flushPromises()

    const moreButton = wrapper
      .findAll('button')
      .find((button) => button.attributes('title') === 'common.more')

    expect(moreButton).toBeDefined()
    await moreButton!.trigger('click')
    await flushPromises()

    const teleportedButtons = Array.from(document.body.querySelectorAll('button'))
    const disableButton = teleportedButtons.find((button) => button.textContent?.includes('停用代理'))

    expect(disableButton).toBeDefined()
    disableButton!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()

    expect(updateProxy).toHaveBeenCalledWith(1, { status: 'inactive' })
    expect(showSuccess).toHaveBeenCalledWith('代理已停用')
    expect(listProxies.mock.calls.length).toBeGreaterThanOrEqual(2)
  })

  it('disables row-level actions while a more-menu async action is in flight', async () => {
    let resolveUpdate: (() => void) | null = null
    updateProxy.mockImplementationOnce(
      () =>
        new Promise<void>((resolve) => {
          resolveUpdate = resolve
        })
    )

    const wrapper = mountProxiesView()
    await flushPromises()

    const moreButton = wrapper
      .findAll('button')
      .find((button) => button.attributes('title') === 'common.more')

    expect(moreButton).toBeDefined()
    await moreButton!.trigger('click')
    await flushPromises()

    const disableButton = Array.from(document.body.querySelectorAll('button')).find((button) =>
      button.textContent?.includes('停用代理')
    ) as HTMLButtonElement | undefined

    expect(disableButton).toBeDefined()

    disableButton!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()

    expect(updateProxy).toHaveBeenCalledTimes(1)

    expect(
      Array.from(document.body.querySelectorAll('button')).some((button) =>
        button.textContent?.includes('停用代理')
      )
    ).toBe(false)

    disableButton!.dispatchEvent(new MouseEvent('click', { bubbles: true }))
    await flushPromises()
    expect(updateProxy).toHaveBeenCalledTimes(1)

    resolveUpdate?.()
    await flushPromises()
  })
})
