import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import Toggle from '../Toggle.vue'

describe('Toggle', () => {
  it('treats a missing model value as disabled', () => {
    const wrapper = mount(Toggle)

    expect(wrapper.attributes('aria-checked')).toBe('false')
    expect(wrapper.classes()).toContain('bg-gray-200')
  })
})
