import { mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { describe, expect, it } from 'vitest'
import RoutingCandidateOptions from '../RoutingCandidateOptions.vue'

const options = [
  { value: 6, label: 'GPT Plus', description: 'OpenAI standard' },
  { value: 7, label: 'GPT Pro', description: 'OpenAI standard' }
]
const setup = (disabled = false) => mount(RoutingCandidateOptions, {
  props: { options, disabled },
  global: { plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false })] }
})

describe('RoutingCandidateOptions', () => {
  it('shows all options directly without search and emits the selected ID', async () => {
    const wrapper = setup()
    expect(wrapper.find('input[type="search"]').exists()).toBe(false)
    expect(wrapper.findAll('input[type="checkbox"]')).toHaveLength(2)
    await wrapper.get('input[aria-label="GPT Pro"]').setValue(true)
    expect(wrapper.emitted('select')).toEqual([[7]])
    wrapper.unmount()
  })

  it('keeps the list controlled by the parent and removes admitted options', async () => {
    const wrapper = setup()
    await wrapper.get('input[aria-label="GPT Plus"]').setValue(true)
    await wrapper.setProps({ options: options.slice(1) })
    expect(wrapper.find('input[aria-label="GPT Plus"]').exists()).toBe(false)
    expect(wrapper.find('input[aria-label="GPT Pro"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('prevents selection while loading or at the candidate limit', async () => {
    const wrapper = setup(true)
    for (const checkbox of wrapper.findAll('input[type="checkbox"]')) {
      expect(checkbox.attributes('disabled')).toBeDefined()
      await checkbox.trigger('change')
    }
    expect(wrapper.emitted('select')).toBeUndefined()
    wrapper.unmount()
  })
})
