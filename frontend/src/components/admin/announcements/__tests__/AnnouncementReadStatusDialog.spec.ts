import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AnnouncementReadStatusDialog from '../AnnouncementReadStatusDialog.vue'

const { getReadStatus, showError } = vi.hoisted(() => ({
  getReadStatus: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    announcements: {
      getReadStatus,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/composables/usePersistedPageSize', () => ({
  getPersistedPageSize: () => 20,
}))

const BaseDialogStub = {
  props: ['show', 'title', 'width'],
  emits: ['close'],
  template: '<div><slot /><slot name="footer" /></div>',
}

const DataTableStub = {
  props: ['columns', 'data', 'defaultSortKey'],
  template: `
    <div>
      <div data-test="columns">{{ columns.map((column) => column.key).join(',') }}</div>
      <div data-test="default-sort">{{ defaultSortKey }}</div>
      <div v-for="row in data" :key="row.user_id">
        <slot name="cell-username" :row="row" :value="row.username" />
      </div>
    </div>
  `,
}

describe('AnnouncementReadStatusDialog', () => {
  beforeEach(() => {
    getReadStatus.mockReset()
    showError.mockReset()
    vi.useFakeTimers()
  })

  it('closes by aborting active requests and clearing debounced reloads', async () => {
    let activeSignal: AbortSignal | undefined
    getReadStatus.mockImplementation(async (...args: any[]) => {
      activeSignal = args[4]?.signal
      return new Promise(() => {})
    })

    const wrapper = mount(AnnouncementReadStatusDialog, {
      props: {
        show: false,
        announcementId: 1,
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          DataTable: true,
          Pagination: true,
          Icon: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(getReadStatus).toHaveBeenCalledTimes(1)
    expect(activeSignal?.aborted).toBe(false)

    const setupState = (wrapper.vm as any).$?.setupState
    setupState.search = 'alice'
    setupState.handleSearch()

    setupState.handleClose()
    await flushPromises()

    expect(activeSignal?.aborted).toBe(true)
    expect(wrapper.emitted('close')).toHaveLength(1)

    vi.advanceTimersByTime(350)
    await flushPromises()

    expect(getReadStatus).toHaveBeenCalledTimes(1)
  })

  it('merges email and username into one username-first user column', async () => {
    getReadStatus.mockResolvedValue({
      items: [
        { user_id: 1, username: 'Alice', email: 'alice@example.com', balance: 1, eligible: true },
        { user_id: 2, username: '', email: 'fallback@example.com', balance: 2, eligible: true },
      ],
      total: 2,
      page: 1,
      page_size: 20,
      pages: 1,
    })

    const wrapper = mount(AnnouncementReadStatusDialog, {
      props: { show: false, announcementId: 1 },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          DataTable: DataTableStub,
          Pagination: true,
          Icon: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.get('[data-test="columns"]').text().split(',')).toEqual([
      'username', 'balance', 'eligible', 'read_at',
    ])
    expect(wrapper.get('[data-test="default-sort"]').text()).toBe('username')
    expect(wrapper.text()).toContain('Alice')
    expect(wrapper.text()).toContain('fallback@example.com')
    expect(wrapper.text()).not.toContain('alice@example.com')
    expect(getReadStatus).toHaveBeenCalledWith(
      1,
      1,
      20,
      expect.objectContaining({ sort_by: 'username', sort_order: 'asc' }),
      expect.any(Object),
    )
  })
})
