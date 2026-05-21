import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const labels: Record<string, string> = {
          'admin.accounts.bulkActions.selected': `${params?.count} account(s) selected`,
          'admin.accounts.bulkActions.selectCurrentPage': 'Select this page',
          'admin.accounts.bulkActions.clear': 'Clear selection',
          'admin.accounts.bulkActions.edit': 'Bulk Edit',
          'admin.accounts.bulkActions.delete': 'Bulk Delete',
          'admin.accounts.bulkActions.test': 'Test Selected',
          'admin.accounts.bulkActions.testAllUngrouped': 'Test Ungrouped',
          'admin.accounts.bulkActions.testUngroupedPrefix': 'Test count',
          'admin.accounts.bulkActions.testUngroupedCountHint': `Total ${params?.total} accounts`,
          'admin.accounts.bulkActions.resetStatus': 'Reset Status',
          'admin.accounts.bulkActions.refreshToken': 'Refresh Token',
          'admin.accounts.bulkActions.setPrivacy': 'Set Privacy',
          'admin.accounts.bulkActions.clearPrivacy': 'Clear Privacy',
          'admin.accounts.batchTest.loadingTargets': 'Preparing Targets',
          'admin.accounts.bulkActions.enableScheduling': 'Enable Scheduling',
          'admin.accounts.bulkActions.disableScheduling': 'Disable Scheduling',
          'admin.accounts.bulkEdit.title': 'Bulk Edit Accounts'
        }
        return labels[key] ?? key
      }
    })
  }
})

function mountBar(selectedIds: number[], extraProps: Record<string, unknown> = {}) {
  return mount(AccountBulkActionsBar, {
    props: { selectedIds, ...extraProps }
  })
}

describe('AccountBulkActionsBar', () => {
  it('shows only the selected-account edit action when rows are selected', () => {
    const wrapper = mountBar([1, 2])

    expect(wrapper.get('[data-testid="account-bulk-edit-selected"]').text()).toBe('Bulk Edit')
    expect(wrapper.text()).toContain('Set Privacy')
    expect(wrapper.text()).toContain('Clear Privacy')
  })

  it('shows no edit action when no rows are selected', () => {
    const wrapper = mountBar([])

    expect(wrapper.find('[data-testid="account-bulk-edit-selected"]').exists()).toBe(false)
  })

  it('shows the ungrouped one-click test action when requested', () => {
    const wrapper = mountBar([], {
      showTestAllUngrouped: true,
      ungroupedTestLimit: 50,
      ungroupedTotalCount: 200
    })

    expect(wrapper.get('[data-testid="account-bulk-test-all-ungrouped"]').text()).toBe('Test Ungrouped')
    expect((wrapper.get('[data-testid="account-bulk-test-all-ungrouped-limit"]').element as HTMLInputElement).value).toBe('50')
  })

  it('emits set-privacy when clicking the batch set privacy action', async () => {
    const wrapper = mountBar([1, 2])

    await wrapper.get('[data-testid="account-bulk-set-privacy"]').trigger('click')

    expect(wrapper.emitted('set-privacy')).toHaveLength(1)
  })

  it('emits clear-privacy when clicking the batch clear privacy action', async () => {
    const wrapper = mountBar([1, 2])

    await wrapper.get('[data-testid="account-bulk-clear-privacy"]').trigger('click')

    expect(wrapper.emitted('clear-privacy')).toHaveLength(1)
  })
})
