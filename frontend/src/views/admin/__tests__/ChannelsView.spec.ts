import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import ChannelsView from '../ChannelsView.vue'

const {
  listChannels,
  updateChannel,
  createChannel,
  removeChannel,
  syncPricingModels,
  getAllGroups,
  getAccountById,
  getWebSearchEmulationConfig,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  listChannels: vi.fn(),
  updateChannel: vi.fn(),
  createChannel: vi.fn(),
  removeChannel: vi.fn(),
  syncPricingModels: vi.fn(),
  getAllGroups: vi.fn(),
  getAccountById: vi.fn(),
  getWebSearchEmulationConfig: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    channels: {
      list: listChannels,
      update: updateChannel,
      create: createChannel,
      remove: removeChannel,
      syncPricingModels,
    },
    groups: {
      getAll: getAllGroups,
    },
    settings: {
      getWebSearchEmulationConfig,
    },
    accounts: {
      getById: getAccountById,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
    showWarning: vi.fn(),
    showInfo: vi.fn(),
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

describe('ChannelsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listChannels.mockResolvedValue({ items: [], total: 0, pages: 1 })
    getAllGroups.mockResolvedValue([])
    getAccountById.mockResolvedValue({ id: 1, name: 'acc-1' })
    getWebSearchEmulationConfig.mockResolvedValue({ enabled: false, providers: [] })
    updateChannel.mockResolvedValue({})
    createChannel.mockResolvedValue({})
  })

  it('submits an empty string when clearing a channel description', async () => {
    const wrapper = mount(ChannelsView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
          DataTable: true,
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          ConfirmDialog: true,
          EmptyState: true,
          Select: true,
          Toggle: true,
          Pagination: true,
          Icon: true,
          PlatformIcon: true,
          ModelTagInput: true,
          GroupBadge: true,
          GroupOptionItem: true,
          ProxySelector: true,
          SearchInput: true,
        },
      },
    })

    await flushPromises()

    const setupState = (wrapper.vm as any).$?.setupState ?? (wrapper.vm as any)
    setupState.editingChannel = { id: 7 }
    setupState.form.name = 'Channel A'
    setupState.form.description = '   '
    setupState.form.status = 'active'
    setupState.form.platforms = []

    await setupState.handleSubmit()

    expect(updateChannel).toHaveBeenCalledWith(
      7,
      expect.objectContaining({
        description: '',
      }),
    )
  })
})
