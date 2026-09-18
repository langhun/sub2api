import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import CapacityBadge from '../CapacityBadge.vue'

describe('CapacityBadge', () => {
  it('renders a stable placeholder when the limit is unavailable', () => {
    const wrapper = mount(CapacityBadge, {
      props: {
        colorClass: 'text-gray-600',
        current: 0,
      },
    })

    expect(wrapper.text()).toContain('0/—')
  })
})
