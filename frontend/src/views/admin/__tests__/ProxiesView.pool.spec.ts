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
        <slot name="cell-status" :row="row" :value="row.status" />
      </div>
    </div>
  `
}

const SelectStub = {
  props: ['modelValue', 'options'],
  template: '<div data-test="select-stub"></div>'
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

  it('renders pool membership and health badges from proxy runtime state', async () => {
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
    expect(wrapper.text()).toContain('admin.proxies.healthCooldown')
    expect(wrapper.text()).toContain('admin.proxies.failoverSwitchCount')
  })
})
