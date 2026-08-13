import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import PaymentProviderDialog from '@/components/payment/PaymentProviderDialog.vue'
import type { ProviderInstance } from '@/types/payment'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

function mountDialog(provider: ProviderInstance) {
  return mount(PaymentProviderDialog, {
    props: {
      show: true,
      saving: false,
      editing: provider,
      allKeyOptions: [{ value: 'bepusdt', label: 'BEpusdt' }],
      enabledKeyOptions: [{ value: 'bepusdt', label: 'BEpusdt' }],
      allPaymentTypes: [{ value: 'bepusdt', label: 'USDT' }],
      redirectLabel: 'Redirect',
    },
    global: {
      stubs: {
        BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
        Select: { props: ['modelValue', 'options', 'disabled'], template: '<div />' },
        ToggleSwitch: { template: '<div />' },
      },
    },
  })
}

describe('BEpusdt callback URL', () => {
  it('uses the configured notification base instead of the browser origin', async () => {
    const provider: ProviderInstance = {
      id: 1,
      provider_key: 'bepusdt',
      name: 'BEpusdt',
      config: {
        apiBase: 'https://bepusdt.daigua.icu',
        notifyUrl: 'https://api.daigua.icu/api/v1/payment/webhook/bepusdt',
        returnUrl: 'https://api.daigua.icu/payment/result',
      },
      supported_types: ['bepusdt'],
      enabled: true,
      payment_mode: 'redirect',
      refund_enabled: false,
      allow_user_refund: false,
      limits: '',
      sort_order: 0,
    }
    const wrapper = mountDialog(provider)

    ;(wrapper.vm as unknown as { loadProvider: (value: ProviderInstance) => void }).loadProvider(provider)
    await nextTick()

    expect(wrapper.text()).toContain('https://api.daigua.icu/api/v1/payment/webhook/bepusdt')
    expect(wrapper.text()).not.toContain(`${window.location.origin}/api/v1/payment/webhook/bepusdt`)
  })

  it('normalizes a trailing slash when saving callback URLs', async () => {
    const provider: ProviderInstance = {
      id: 1,
      provider_key: 'bepusdt',
      name: 'BEpusdt',
      config: {
        apiBase: 'https://bepusdt.daigua.icu',
        notifyUrl: 'https://api.daigua.icu/api/v1/payment/webhook/bepusdt',
        returnUrl: 'https://api.daigua.icu/payment/result',
      },
      supported_types: ['bepusdt'],
      enabled: true,
      payment_mode: 'redirect',
      refund_enabled: false,
      allow_user_refund: false,
      limits: '',
      sort_order: 0,
    }
    const wrapper = mountDialog(provider)
    ;(wrapper.vm as unknown as { loadProvider: (value: ProviderInstance) => void }).loadProvider(provider)
    await nextTick()

    const callbackBaseInputs = wrapper.findAll('input[type="text"]')
      .filter(input => (input.element as HTMLInputElement).value === 'https://api.daigua.icu')
    expect(callbackBaseInputs).toHaveLength(2)
    await callbackBaseInputs[0].setValue('https://api.daigua.icu/')
    await wrapper.find('form').trigger('submit.prevent')

    const payload = wrapper.emitted('save')?.[0]?.[0] as { config: Record<string, string> }
    expect(payload.config.notifyUrl).toBe('https://api.daigua.icu/api/v1/payment/webhook/bepusdt')
  })
})
