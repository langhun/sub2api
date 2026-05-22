import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserAttributesConfigModal from '../UserAttributesConfigModal.vue'

const {
  listDefinitions,
  updateDefinition,
  createDefinition,
  showError,
  showSuccess,
} = vi.hoisted(() => ({
  listDefinitions: vi.fn(),
  updateDefinition: vi.fn(),
  createDefinition: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    userAttributes: {
      listDefinitions,
      updateDefinition,
      createDefinition,
      deleteDefinition: vi.fn(),
      reorderDefinitions: vi.fn(),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
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

describe('UserAttributesConfigModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    listDefinitions.mockResolvedValue([])
    updateDefinition.mockResolvedValue({})
    createDefinition.mockResolvedValue({})
  })

  it('submits empty description and placeholder strings when clearing an existing definition', async () => {
    const wrapper = mount(UserAttributesConfigModal, {
      props: {
        show: true,
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: true,
          Icon: true,
          ConfirmDialog: true,
        },
      },
    })

    await flushPromises()

    const setupState = (wrapper.vm as any).$?.setupState ?? (wrapper.vm as any)
    setupState.editingAttribute = {
      id: 12,
      key: 'nickname',
      name: 'Nickname',
      type: 'text',
      description: 'Old description',
      placeholder: 'Old placeholder',
      required: false,
      enabled: true,
      display_order: 1,
      options: [],
    }
    setupState.form.key = 'nickname'
    setupState.form.name = 'Nickname'
    setupState.form.type = 'text'
    setupState.form.description = '   '
    setupState.form.placeholder = '   '
    setupState.form.required = false
    setupState.form.enabled = true

    await setupState.handleSave()

    expect(updateDefinition).toHaveBeenCalledWith(
      12,
      expect.objectContaining({
        description: '',
        placeholder: '',
      }),
    )
  })
})
