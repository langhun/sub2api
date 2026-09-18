import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import PaymentMethodSelector from '@/components/payment/PaymentMethodSelector.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
  }),
}))

describe('PaymentMethodSelector', () => {
  it('wraps large custom method collections without letting labels widen the selector', () => {
    const methods = Array.from({ length: 12 }, (_, index) => ({
      type: `custom_${index}`,
      display_name: `CUSTOM_PAYMENT_METHOD_${index}`,
      fee_rate: 0,
      available: true,
    }))

    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'custom_0',
        methods,
      },
    })

    const grid = wrapper.get('[data-testid="payment-method-grid"]')
    expect(grid.classes()).toContain('payment-method-grid')

    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(methods.length)
    expect(buttons.every(button => button.classes().includes('min-w-0'))).toBe(true)
    expect(buttons.every((button, index) => button.attributes('title') === methods[index].display_name)).toBe(true)
    expect(wrapper.findAll('[data-testid="payment-method-label"]').every(label => label.classes().includes('whitespace-normal'))).toBe(true)
    expect(wrapper.findAll('[data-testid="payment-method-label"]').every(label => label.classes().includes('break-words'))).toBe(true)
  })

  it('can hide the repeated field label when the parent section already provides one', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'wxpay',
        showLabel: false,
        methods: [{ type: 'wxpay', fee_rate: 0, available: true }],
      },
    })

    expect(wrapper.find('label').exists()).toBe(false)
    expect(wrapper.get('button').text()).toContain('wxpay')
  })

  it('shows the configured display name for custom EasyPay methods', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'ldc',
        methods: [{ type: 'ldc', display_name: 'LDC Pay', fee_rate: 0, available: true }],
      },
    })

    expect(wrapper.text()).toContain('LDC Pay')
    expect(wrapper.text()).not.toContain('ldc')
    expect(wrapper.text()).not.toContain('payment.methods.ldc')
  })

  it('uses the generic selected style for custom methods that contain built-in names', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'card_alipay',
        methods: [{ type: 'card_alipay', display_name: 'Card Pay', fee_rate: 0, available: true }],
      },
    })

    const button = wrapper.get('button')
    expect(button.classes()).toContain('border-primary-500')
    expect(button.classes()).not.toContain('border-[#02A9F1]')
  })

  it('keeps a method visible and explains its minimum amount when disabled', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'alipay',
        methods: [{
          type: 'minimum-limited',
          display_name: 'Limited Pay',
          fee_rate: 0,
          available: false,
          unavailable_reason: 'minimum',
          minimum_amount: 20,
          currency: 'CNY',
        }],
      },
    })

    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-testid="payment-method-limit-hint"]').text()).toContain('¥20.00')
  })

  it('keeps limit information present across amount changes and prevents disabled selection', async () => {
    const method = { type: 'alipay', fee_rate: 0, available: false, minimum_amount: 20, maximum_amount: 500, unavailable_reason: 'minimum' as const }
    const wrapper = mount(PaymentMethodSelector, { props: { selected: 'alipay', methods: [method] } })
    const hints = wrapper.findAll('[data-testid="payment-method-limit-hint"]').map(hint => hint.text())
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('select')).toBeUndefined()
    expect(wrapper.get('button').attributes('aria-pressed')).toBe('false')
    await wrapper.setProps({ methods: [{ ...method, available: true, unavailable_reason: undefined }] })
    expect(wrapper.findAll('[data-testid="payment-method-limit-hint"]').map(hint => hint.text())).toEqual(hints)
    expect(wrapper.get('button').attributes('aria-pressed')).toBe('true')
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('select')).toEqual([['alipay']])
  })

  it('reserves detail space so changing amount limits does not resize method cards', () => {
    const wrapper = mount(PaymentMethodSelector, {
      props: {
        selected: 'alipay',
        methods: [{ type: 'alipay', fee_rate: 0, available: true }],
      },
    })

    expect(wrapper.get('button').classes()).toContain('min-h-28')
    expect(wrapper.get('.payment-method-details').classes()).toContain('min-h-12')
  })
})
