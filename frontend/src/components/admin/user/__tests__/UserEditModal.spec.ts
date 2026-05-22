import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UserEditModal from '../UserEditModal.vue'

const {
  updateUser,
  updateUserAttributeValues,
  showError,
  showSuccess,
  copyToClipboard,
} = vi.hoisted(() => ({
  updateUser: vi.fn(),
  updateUserAttributeValues: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  copyToClipboard: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      update: updateUser,
    },
    userAttributes: {
      updateUserAttributeValues,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError,
    showSuccess,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard,
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

describe('UserEditModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    updateUser.mockResolvedValue({})
    updateUserAttributeValues.mockResolvedValue({ message: 'ok' })
  })

  it('submits an empty custom attributes object so cleared attributes are persisted', async () => {
    const wrapper = mount(UserEditModal, {
      props: {
        show: true,
        user: {
          id: 7,
          email: 'user@example.com',
          username: 'user',
          notes: '',
          concurrency: 1,
          rpm_limit: 0,
        } as any,
      },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          UserAttributeForm: { template: '<div />' },
          Icon: true,
        },
      },
    })

    await wrapper.get('form#edit-user-form').trigger('submit')
    await flushPromises()

    expect(updateUser).toHaveBeenCalledWith(
      7,
      expect.objectContaining({
        email: 'user@example.com',
      }),
    )
    expect(updateUserAttributeValues).toHaveBeenCalledWith(7, {})
    expect(showSuccess).toHaveBeenCalled()
  })
})
