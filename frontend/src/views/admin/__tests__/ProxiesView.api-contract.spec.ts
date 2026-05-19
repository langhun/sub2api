import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ProxiesView from '../ProxiesView.vue'

const {
  listProxies,
  createProxy,
  getAllGroups,
  listProxySubscriptions,
  updatePoolMembership,
  clearCooldown
} = vi.hoisted(() => ({
  listProxies: vi.fn(),
  createProxy: vi.fn(),
  getAllGroups: vi.fn(),
  listProxySubscriptions: vi.fn(),
  updatePoolMembership: vi.fn(),
  clearCooldown: vi.fn()
}))

const appStore = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showInfo: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      list: listProxies,
      create: createProxy,
      batchCreate: vi.fn(),
      update: vi.fn(),
      delete: vi.fn(),
      batchDelete: vi.fn(),
      unassignAccounts: vi.fn(),
      getProxyAccounts: vi.fn(),
      testProxy: vi.fn(),
      checkProxyQuality: vi.fn(),
      exportData: vi.fn(),
      importData: vi.fn(),
      updatePoolMembership,
      clearCooldown
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
  useAppStore: () => appStore
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
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

const ProxiesToolbarStub = {
  props: ['searchQuery', 'filters'],
  emits: [
    'update:search-query',
    'update:filters',
    'create-proxy',
    'reload-proxies',
    'set-tab'
  ],
  template: `
    <div data-test="toolbar-stub">
      <button data-test="toolbar-search" @click="$emit('update:search-query', 'edge node')">search</button>
      <button
        data-test="toolbar-filters"
        @click="$emit('update:filters', { protocol: 'socks5', status: 'inactive', runtime_status: 'challenge' })"
      >
        filters
      </button>
      <button data-test="toolbar-reload" @click="$emit('reload-proxies')">reload</button>
      <button data-test="toolbar-create" @click="$emit('create-proxy')">create</button>
    </div>
  `
}

const TablePageLayoutStub = {
  template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
}

const DataTableStub = {
  props: ['data', 'columns'],
  template: `
    <div data-test="data-table-stub" :data-columns="columns.map(column => column.key).join(',')">
      <div v-for="row in data" :key="row.id">{{ row.name }}</div>
    </div>
  `
}

const BaseDialogStub = {
  props: ['show', 'title'],
  template: `
    <div v-if="show" data-test="dialog-stub">
      <slot />
      <slot name="footer" />
    </div>
  `
}

const SelectStub = {
  props: ['modelValue', 'options', 'placeholder'],
  emits: ['update:modelValue', 'change'],
  template: '<div data-test="select-stub" :data-placeholder="placeholder">{{ modelValue }}</div>'
}

const mountView = () =>
  mount(ProxiesView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: TablePageLayoutStub,
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
        Select: SelectStub,
        Icon: true,
        Teleport: true
      }
    }
  })

describe('admin ProxiesView API contracts', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    localStorage.clear()

    listProxies.mockReset()
    createProxy.mockReset()
    getAllGroups.mockReset()
    listProxySubscriptions.mockReset()
    updatePoolMembership.mockReset()
    clearCooldown.mockReset()
    appStore.showError.mockReset()
    appStore.showSuccess.mockReset()
    appStore.showInfo.mockReset()

    listProxies.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'edge-1',
          protocol: 'http',
          host: 'proxy.example.com',
          port: 8080,
          username: null,
          password: null,
          status: 'active',
          health_status: 'healthy',
          country: 'United States',
          country_code: 'us',
          city: 'Los Angeles',
          auto_failover_pool_enabled: false,
          created_at: '2026-05-19T00:00:00Z',
          updated_at: '2026-05-19T00:00:00Z'
        }
      ],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    createProxy.mockResolvedValue({
      id: 2,
      name: 'new-edge',
      protocol: 'socks5',
      host: 'new.example.com',
      port: 1080,
      username: 'demo-user',
      password: 'demo-pass',
      status: 'active',
      health_status: 'healthy',
      country: '',
      country_code: '',
      city: '',
      auto_failover_pool_enabled: true,
      created_at: '2026-05-19T00:00:00Z',
      updated_at: '2026-05-19T00:00:00Z'
    })
    getAllGroups.mockResolvedValue([])
    listProxySubscriptions.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 100,
      pages: 0
    })
    updatePoolMembership.mockResolvedValue({ updated: 0, enabled: true })
    clearCooldown.mockResolvedValue({ cleared: 0 })
  })

  it('sends runtime_status and search through the page-level proxy list contract', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listProxies).toHaveBeenCalledTimes(1)
    expect(listProxies).toHaveBeenNthCalledWith(
      1,
      1,
      20,
      expect.objectContaining({
        protocol: undefined,
        status: undefined,
        runtime_status: undefined,
        search: undefined,
        sort_by: 'id',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )

    await wrapper.get('[data-test="toolbar-filters"]').trigger('click')
    await wrapper.get('[data-test="toolbar-search"]').trigger('click')
    vi.advanceTimersByTime(300)
    await flushPromises()

    expect(listProxies).toHaveBeenCalledTimes(2)
    expect(listProxies).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        protocol: 'socks5',
        status: 'inactive',
        runtime_status: 'challenge',
        search: 'edge node',
        sort_by: 'id',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )
  })

  it('reload button reuses the current page-level filters when requesting proxies', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="toolbar-filters"]').trigger('click')
    await wrapper.get('[data-test="toolbar-search"]').trigger('click')
    vi.advanceTimersByTime(300)
    await flushPromises()

    await wrapper.get('[data-test="toolbar-reload"]').trigger('click')
    await flushPromises()

    expect(listProxies).toHaveBeenCalledTimes(3)
    expect(listProxies).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        protocol: 'socks5',
        status: 'inactive',
        runtime_status: 'challenge',
        search: 'edge node'
      }),
      expect.any(Object)
    )
  })

  it('submits create-proxy from the toolbar with the normalized payload expected by the API', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[data-test="toolbar-create"]').trigger('click')
    await flushPromises()

    const textInputs = wrapper.findAll('input[type="text"]')
    expect(textInputs.length).toBeGreaterThanOrEqual(3)

    await textInputs[0].setValue('  new-edge  ')
    await textInputs[1].setValue('  new.example.com  ')
    await textInputs[2].setValue('  demo-user  ')

    const numberInput = wrapper.get('input[type="number"]')
    await numberInput.setValue('1080')

    const passwordInput = wrapper.get('input[type="password"]')
    await passwordInput.setValue('  demo-pass  ')

    const poolToggle = wrapper.get('input[type="checkbox"]')
    await poolToggle.setValue(true)

    await wrapper.get('form#create-proxy-form').trigger('submit')
    await flushPromises()

    expect(createProxy).toHaveBeenCalledTimes(1)
    expect(createProxy).toHaveBeenCalledWith({
      name: 'new-edge',
      protocol: 'http',
      host: 'new.example.com',
      port: 1080,
      username: 'demo-user',
      password: 'demo-pass',
      auto_failover_pool_enabled: true
    })

    expect(appStore.showSuccess).toHaveBeenCalledWith('admin.proxies.proxyCreated')
    expect(listProxies).toHaveBeenCalledTimes(2)
  })
})
