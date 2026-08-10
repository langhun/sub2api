import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Account } from '@/types'

const { getOpenAIRTMock, replaceOpenAIRTMock, showSuccessMock } = vi.hoisted(() => ({
  getOpenAIRTMock: vi.fn(),
  replaceOpenAIRTMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('../api', () => ({
  getOpenAIRT: getOpenAIRTMock,
  replaceOpenAIRT: replaceOpenAIRTMock,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: showSuccessMock,
  }),
}))

import OpenAIRTModal from '../OpenAIRTModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})

const account: Account = {
  id: 1,
  name: 'OpenAI OAuth',
  platform: 'openai',
  type: 'oauth',
  credentials: { auth_mode: 'oauth' },
  credentials_status: { has_refresh_token: true },
  proxy_id: null,
  concurrency: 3,
  status: 'active',
}

describe('OpenAIRTModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getOpenAIRTMock.mockResolvedValue({
      account_id: account.id,
      refresh_token: 'rt-old-secret',
      masked_refresh_token: 'rt-old...cret',
    })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
  })

  it('loads the current RT and copies it with the Clipboard API', async () => {
    const wrapper = mount(OpenAIRTModal, {
      props: { show: true, account },
      global: { stubs: { BaseDialog: BaseDialogStub } },
    })
    await flushPromises()

    expect(getOpenAIRTMock).toHaveBeenCalledWith(account.id)
    expect(wrapper.get<HTMLInputElement>('[data-testid="openai-rt-old-token"]').element.value).toBe('rt-old-secret')

    const copyButton = wrapper.findAll('button').find((button) => button.text() === 'admin.openaiRt.copy')
    expect(copyButton).toBeDefined()
    await copyButton!.trigger('click')

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('rt-old-secret')
    expect(replaceOpenAIRTMock).not.toHaveBeenCalled()
  })

  it('replaces the RT and clears sensitive state after success', async () => {
    replaceOpenAIRTMock.mockResolvedValue({ account, masked_refresh_token: 'rt-new...cret' })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const wrapper = mount(OpenAIRTModal, {
      props: { show: true, account },
      global: { stubs: { BaseDialog: BaseDialogStub } },
    })
    await flushPromises()

    await wrapper.get<HTMLInputElement>('[data-testid="openai-rt-new-token"]').setValue('rt-new-secret')
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()

    expect(replaceOpenAIRTMock).toHaveBeenCalledWith(account.id, 'rt-new-secret')
    expect(showSuccessMock).toHaveBeenCalledWith('admin.openaiRt.replaceSuccess')
    expect(wrapper.emitted('updated')).toEqual([[account]])
    expect(wrapper.get<HTMLInputElement>('[data-testid="openai-rt-old-token"]').element.value).toBe('')
  })
})
