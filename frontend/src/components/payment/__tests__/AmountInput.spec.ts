import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import AmountInput from '@/components/payment/AmountInput.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => key,
  }),
}))

enableAutoUnmount(afterEach)

function mountInput(value: number | null = null) {
  return mount(AmountInput, { props: { modelValue: value } })
}

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

  it('restores the displayed amount when pasted text is invalid', async () => {
    const wrapper = mount(AmountInput, { props: { modelValue: 20 } })
    const input = wrapper.get<HTMLInputElement>('[data-testid="amount-input"]')
    for (const value of ['20oops', '12.345', '9'.repeat(310)]) {
      await input.setValue(value)
      expect(input.element.value).toBe('20')
    }
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
    await wrapper.setProps({ modelValue: null })
    expect(input.element.value).toBe('')
  })

  it('exposes the selected amount and validation state to assistive technology', async () => {
    const wrapper = mount(AmountInput, { props: { modelValue: 20, error: 'Minimum is 50' } })
    expect(wrapper.get('[data-testid="quick-amount-20"]').attributes('aria-pressed')).toBe('true')
    expect(wrapper.get('[data-testid="amount-input"]').attributes('aria-invalid')).toBe('true')
    expect(wrapper.get('[role="status"]').text()).toBe('Minimum is 50')
    await wrapper.setProps({ error: '' })
    expect(wrapper.get('[role="status"]').text()).toBe('')
    expect(wrapper.get('[data-testid="amount-input"]').attributes('aria-invalid')).toBe('false')
  })

  it.each(['10abc', '10.555', '-10', '1e2'])('restores the accepted amount after rejecting %s', async (value) => {
    const wrapper = mountInput(10)
    const input = wrapper.get('input')
    await input.setValue(value)
    expect((input.element as HTMLInputElement).value).toBe('10')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('restores the last typed amount rather than a stale prop', async () => {
    const wrapper = mountInput()
    const input = wrapper.get('input')
    await input.setValue('12.50')
    await input.setValue('12.500')
    expect((input.element as HTMLInputElement).value).toBe('12.50')
    expect(wrapper.emitted('update:modelValue')).toEqual([[12.5]])
  })

  it('preserves decimal editing and allows clearing the amount', async () => {
    const wrapper = mountInput()
    const input = wrapper.get('input')
    for (const value of ['0', '0.', '0.5', '0.50', '']) await input.setValue(value)
    expect(wrapper.emitted('update:modelValue')).toEqual([[null], [null], [0.5], [0.5], [null]])
    expect((input.element as HTMLInputElement).value).toBe('')
  })
})
