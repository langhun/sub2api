import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AmountInput from '@/components/payment/AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

describe('AmountInput', () => {
  it('shows the selected payment currency in the input and quick amounts', () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: 20,
        amounts: [10, 20],
        currency: 'CNY',
      },
    })

    expect(wrapper.get('[data-testid="amount-input"]').element).toHaveProperty('value', '20')
    expect(wrapper.text()).toContain('CNY')
    expect(wrapper.get('[data-testid="quick-amount-10"]').text()).toBe('¥10')
  })

  it('emits preset and decimal custom amounts without changing validation semantics', async () => {
    const wrapper = mount(AmountInput, {
      props: {
        modelValue: null,
        amounts: [10],
        currency: 'USD',
      },
    })

    await wrapper.get('[data-testid="quick-amount-10"]').trigger('click')
    await wrapper.get('[data-testid="amount-input"]').setValue('12.34')

    expect(wrapper.emitted('update:modelValue')).toEqual([[10], [12.34]])
  })
})
