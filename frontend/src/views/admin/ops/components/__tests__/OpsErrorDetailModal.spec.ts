import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import OpsErrorDetailModal from '../OpsErrorDetailModal.vue'

const { getRequestErrorDetail, listRequestErrorUpstreamErrors, showError } = vi.hoisted(() => ({
  getRequestErrorDetail: vi.fn(),
  listRequestErrorUpstreamErrors: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRequestErrorDetail,
    getUpstreamErrorDetail: vi.fn(),
    listRequestErrorUpstreamErrors,
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({ showError }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const BaseDialogStub = {
  props: ['show', 'title'],
  template: '<div v-if="show"><slot /></div>',
}

describe('OpsErrorDetailModal', () => {
  it('shows the username instead of a duplicate email for request errors', async () => {
    getRequestErrorDetail.mockResolvedValue({
      id: 1,
      created_at: '2026-07-20T00:00:00Z',
      phase: 'request',
      error_owner: 'client',
      status_code: 400,
      username: 'Detail User',
      user_email: 'detail@example.com',
      user_id: 8,
      error_body: '{}',
      message: 'bad request',
    })
    listRequestErrorUpstreamErrors.mockResolvedValue({ items: [] })

    const wrapper = mount(OpsErrorDetailModal, {
      props: { show: true, errorId: 1, errorType: 'request' },
      global: { stubs: { BaseDialog: BaseDialogStub, Icon: true } },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Detail User')
    expect(wrapper.text()).not.toContain('detail@example.com')
    expect(showError).not.toHaveBeenCalled()
  })
})
