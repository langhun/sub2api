import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import UserEditModal from '../UserEditModal.vue'

const mocks = vi.hoisted(() => ({ update: vi.fn(), showError: vi.fn(), showSuccess: vi.fn() }))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: { update: mocks.update },
    userAttributes: { updateUserAttributeValues: vi.fn() },
  },
}))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: mocks.showError, showSuccess: mocks.showSuccess }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: vi.fn() }) }))
vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: ref('zh-CN') }) }
})
vi.mock('@/components/common/BaseDialog.vue', () => ({
  default: { props: ['show'], template: '<div v-if="show"><slot/><slot name="footer"/></div>' },
}))
vi.mock('@/components/user/UserAttributeForm.vue', () => ({
  default: { props: ['modelValue'], template: '<div />' },
}))

const user = {
  id: 7,
  email: 'risk@example.com',
  username: 'risk-user',
  notes: '',
  role: 'user',
  concurrency: 5,
  rpm_limit: 0,
  game_hall_disabled: true,
}

describe('UserEditModal game hall governance', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mocks.update.mockResolvedValue(user)
  })

  it('loads and saves the per-user game hall disabled flag', async () => {
    const wrapper = mount(UserEditModal, { props: { show: true, user: user as never }, global: { stubs: { Icon: true } } })
    await flushPromises()

    const toggle = wrapper.get('[data-testid="user-game-hall-disabled"]')
    expect(toggle.attributes('aria-checked')).toBe('true')
    await toggle.trigger('click')
    await wrapper.get('#edit-user-form').trigger('submit')
    await flushPromises()

    expect(mocks.update).toHaveBeenCalledWith(7, expect.objectContaining({ game_hall_disabled: false }))
  })
})
