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
          'admin.accounts.bulkActions.resetStatus': 'Reset Status',
          'admin.accounts.bulkActions.refreshToken': 'Refresh Token',
          'admin.accounts.bulkActions.enableScheduling': 'Enable Scheduling',
          'admin.accounts.bulkActions.disableScheduling': 'Disable Scheduling',
          'admin.accounts.bulkEdit.title': 'Bulk Edit Accounts'
        }
        return labels[key] ?? key
      }
    })
  }
})

function mountBar(selectedIds: number[]) {
  return mount(AccountBulkActionsBar, {
    props: { selectedIds }
  })
}

describe('AccountBulkActionsBar', () => {
  it('shows only the selected-account edit action when rows are selected', () => {
    const wrapper = mountBar([1, 2])

    expect(wrapper.get('[data-testid="account-bulk-edit-selected"]').text()).toBe('Bulk Edit')
  })

  it('shows no edit action when no rows are selected', () => {
    const wrapper = mountBar([])

    expect(wrapper.find('[data-testid="account-bulk-edit-selected"]').exists()).toBe(false)
  })
})
