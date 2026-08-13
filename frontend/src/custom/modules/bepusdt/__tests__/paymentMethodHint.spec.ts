import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'
import { bepusdtLocaleMessages } from '../locales'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: { amount?: string }) => {
      if (key === 'payment.methods.bepusdt') return 'USDT'
      if (key === 'payment.minimumToUse') return `最低充值 ${params?.amount ?? ''} 可使用`
      return key
    },
  }),
}))

describe('BEpusdt payment method hint', () => {
  it('shows the configured minimum instead of only marking USDT unavailable', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: '',
        methods: [{
          type: 'bepusdt',
          display_name: 'USDT',
          fee_rate: 0,
          available: false,
          unavailable_reason: 'minimum',
          minimum_amount: 20,
          currency: 'CNY',
        }],
      },
    })

    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="payment-method-label"]').text()).toBe('USDT')
    expect(wrapper.get('[data-testid="payment-method-limit-hint"]').text()).toBe('最低充值 ¥20.00 可使用')
    expect(bepusdtLocaleMessages.zh.payment.minimumToUse).toContain('{amount}')
  })
})
