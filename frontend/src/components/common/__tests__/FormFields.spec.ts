import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import Input from '../Input.vue'
import TextArea from '../TextArea.vue'

describe.each([{ name: 'Input', component: Input, selector: 'input' }, { name: 'TextArea', component: TextArea, selector: 'textarea' }])('$name native attributes', ({ component, selector }) => {
  it('applies validation and accessible attributes to the native field while preserving wrapper layout', async () => {
    const wrapper = mount(component, {
      props: { modelValue: '', label: 'Name', hint: 'Hint' },
      attrs: { maxlength: 5, name: 'name', 'aria-label': 'Display name', class: 'col-span-2' },
    })
    const field = wrapper.get(selector)
    expect(field.attributes('maxlength')).toBe('5')
    expect(field.attributes('name')).toBe('name')
    expect(field.attributes('aria-label')).toBe('Display name')
    expect(wrapper.attributes('maxlength')).toBeUndefined()
    expect(wrapper.classes()).toContain('col-span-2')
    expect(wrapper.get('label').attributes('for')).toBe(field.attributes('id'))
    expect(field.attributes('aria-describedby')).toBe(wrapper.get('p').attributes('id'))
    await wrapper.setProps({ error: 'Invalid' })
    expect(field.attributes('aria-invalid')).toBe('true')
    wrapper.unmount()
  })
})
