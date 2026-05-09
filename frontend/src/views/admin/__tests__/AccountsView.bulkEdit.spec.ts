import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  deleteAccount,
  getAllProxies,
  getAllGroups
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  deleteAccount: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      delete: deleteAccount,
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      toggleSchedulable: vi.fn()
    },
    proxies: {
      getAll: getAllProxies
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

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    token: 'test-token'
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
  props: ['columns', 'data'],
  template: '<div data-test="data-table"></div>'
}

const AccountBulkActionsBarStub = {
  props: ['selectedIds', 'showTestAllUngrouped', 'ungroupedTestLimit'],
  emits: ['edit-selected', 'test-all-ungrouped', 'update:ungrouped-test-limit'],
  template: `
    <div>
      <button data-test="edit-selected" @click="$emit('edit-selected')">edit selected</button>
      <input
        v-if="showTestAllUngrouped"
        data-test="ungrouped-limit"
        :value="ungroupedTestLimit"
        @input="$emit('update:ungrouped-test-limit', Number($event.target.value))"
      />
      <button
        v-if="showTestAllUngrouped"
        data-test="test-all-ungrouped"
        @click="$emit('test-all-ungrouped')"
      >
        test all ungrouped
      </button>
    </div>
  `
}

const BulkEditAccountModalStub = {
  props: ['show'],
  template: '<div data-test="bulk-edit-modal" :data-show="String(show)"></div>'
}

const BatchAccountTestModalStub = {
  props: ['show', 'targets'],
  template: '<div data-test="batch-test-modal" :data-show="String(show)" :data-target-count="String(targets?.length ?? 0)"></div>'
}

describe('admin AccountsView bulk edit scope', () => {
  beforeEach(() => {
    localStorage.clear()

    listAccounts.mockReset()
    listWithEtag.mockReset()
    getBatchTodayStats.mockReset()
    deleteAccount.mockReset()
    getAllProxies.mockReset()
    getAllGroups.mockReset()

    listAccounts.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    listWithEtag.mockResolvedValue({
      notModified: true,
      etag: null,
      data: null
    })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    deleteAccount.mockResolvedValue({ message: 'ok' })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
  })

  it('opens bulk edit in selected mode from the bulk actions dropdown', async () => {
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          BatchAccountTestModal: BatchAccountTestModalStub,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    await wrapper.get('[data-test="edit-selected"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="bulk-edit-modal"]').attributes('data-show')).toBe('true')
  })

  it('opens batch test with all filtered ungrouped accounts', async () => {
    listAccounts.mockResolvedValueOnce({
      items: [
        { id: 1, name: 'ungrouped-1', platform: 'openai', type: 'apikey', groups: [] },
        { id: 2, name: 'ungrouped-2', platform: 'antigravity', type: 'oauth', groups: [] }
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1
    })
    listAccounts.mockResolvedValueOnce({
      items: [
        { id: 1, name: 'ungrouped-1', platform: 'openai', type: 'apikey', groups: [] },
        { id: 2, name: 'ungrouped-2', platform: 'antigravity', type: 'oauth', groups: [] }
      ],
      total: 2,
      page: 1,
      page_size: 100,
      pages: 1
    })

    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          BatchAccountTestModal: BatchAccountTestModalStub,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    ;(wrapper.vm as any).params.group = 'ungrouped'
    await flushPromises()
    await wrapper.get('[data-test="ungrouped-limit"]').setValue('1')

    await wrapper.get('[data-test="test-all-ungrouped"]').trigger('click')
    await flushPromises()

    expect(wrapper.get('[data-test="batch-test-modal"]').attributes('data-show')).toBe('true')
    expect(wrapper.get('[data-test="batch-test-modal"]').attributes('data-target-count')).toBe('1')
    expect(listAccounts).toHaveBeenLastCalledWith(1, 1, expect.any(Object))
  })

  it('queues 401 batch-test accounts for sequential deletion', async () => {
    const wrapper = mount(AccountsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
          AccountTableFilters: { template: '<div></div>' },
          AccountBulkActionsBar: AccountBulkActionsBarStub,
          AccountActionMenu: true,
          ImportDataModal: true,
          ReAuthAccountModal: true,
          AccountTestModal: true,
          BatchAccountTestModal: BatchAccountTestModalStub,
          AccountStatsModal: true,
          ScheduledTestsPanel: true,
          SyncFromCrsModal: true,
          TempUnschedStatusModal: true,
          ErrorPassthroughRulesModal: true,
          TLSFingerprintProfilesModal: true,
          CreateAccountModal: true,
          EditAccountModal: true,
          BulkEditAccountModal: BulkEditAccountModalStub,
          PlatformTypeBadge: true,
          AccountCapacityCell: true,
          AccountStatusIndicator: true,
          AccountTodayStatsCell: true,
          AccountGroupsCell: true,
          AccountUsageCell: true,
          Icon: true
        }
      }
    })

    await flushPromises()
    await (wrapper.vm as any).enqueueBatchTestDelete(4019)
    await (wrapper.vm as any).waitForBatchTestDeleteQueueIdle()

    expect(deleteAccount).toHaveBeenCalledWith(4019)
  })
})
