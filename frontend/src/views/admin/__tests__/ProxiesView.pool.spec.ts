import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ProxiesView from '../ProxiesView.vue'

const {
  listProxies,
  getAllGroups
} = vi.hoisted(() => ({
  listProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      list: listProxies,
      updatePoolMembership: vi.fn(),
      clearCooldown: vi.fn()
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
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

const DataTableStub = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-location" :row="row" :value="row.location" />
        <slot name="cell-status" :row="row" :value="row.status" />
      </div>
    </div>
  `
}

const SelectStub = {
  props: ['modelValue', 'options', 'placeholder'],
  template: '<div data-test="select-stub" :data-placeholder="placeholder"></div>'
}

describe('admin ProxiesView pool state', () => {
  beforeEach(() => {
    listProxies.mockReset()
    getAllGroups.mockReset()

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
  })

  it('renders pool members in a table dialog and supports keyword filtering', async () => {
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
          Select: SelectStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('admin.proxies.poolMembersAction')

    const openPoolButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.proxies.poolMembersAction'))

    expect(openPoolButton).toBeDefined()
    await openPoolButton!.trigger('click')
    await flushPromises()

    expect(wrapper.find('table').exists()).toBe(true)
    expect(wrapper.text()).toContain('pool-a')
    expect(wrapper.text()).toContain('proxy.example.com:8080')
    expect(wrapper.text()).toContain('admin.proxies.healthCooldown')

    const searchInput = wrapper.find('input[placeholder="admin.proxies.poolMembersSearchPlaceholder"]')
    expect(searchInput.exists()).toBe(true)

    await searchInput.setValue('not-found')
    await flushPromises()

    expect(wrapper.text()).toContain('admin.proxies.poolMembersFilteredEmpty')
    expect(wrapper.text()).not.toContain('pool-a')
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
          Select: SelectStub,
          Icon: true
        }
      }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('🇺🇸')
    expect(wrapper.text()).toContain('美国 · 洛杉矶')
    expect(wrapper.find('img').exists()).toBe(false)
  })
})
