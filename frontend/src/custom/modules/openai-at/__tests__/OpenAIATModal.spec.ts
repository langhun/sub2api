import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { Account } from '@/types'

const { getOpenAIATMock, replaceOpenAIATMock, showSuccessMock } = vi.hoisted(() => ({
  getOpenAIATMock: vi.fn(),
  replaceOpenAIATMock: vi.fn(),
  showSuccessMock: vi.fn(),
}))

vi.mock('../api', () => ({
  getOpenAIAT: getOpenAIATMock,
  replaceOpenAIAT: replaceOpenAIATMock,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess: showSuccessMock,
  }),
}))

import OpenAIATModal from '../OpenAIATModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: { type: Boolean, default: false } },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})

const account: Account = {
  id: 1,
  name: 'OpenAI PAT',
  platform: 'openai',
  type: 'oauth',
  credentials: { auth_mode: 'personalAccessToken' },
  proxy_id: null,
  concurrency: 3,
  status: 'active',
}

describe('OpenAIATModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getOpenAIATMock.mockResolvedValue({
      account_id: account.id,
      access_token: 'at-old-secret',
      masked_access_token: 'at-old...cret',
    })
    Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: vi.fn().mockResolvedValue(undefined) },
    })
  })

  it('loads the current AT and copies it with the Clipboard API', async () => {
    const wrapper = mount(OpenAIATModal, {
      props: { show: true, account },
      global: { stubs: { BaseDialog: BaseDialogStub } },
    })
    await flushPromises()

    expect(getOpenAIATMock).toHaveBeenCalledWith(account.id)
    expect(wrapper.get<HTMLInputElement>('[data-testid="openai-at-old-token"]').element.value).toBe('at-old-secret')

    const copyButton = wrapper.findAll('button').find((button) => button.text() === 'admin.openaiAt.copy')
    expect(copyButton).toBeDefined()
    await copyButton!.trigger('click')

    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('at-old-secret')
    expect(replaceOpenAIATMock).not.toHaveBeenCalled()
  })
})
