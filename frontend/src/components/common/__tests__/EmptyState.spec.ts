import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import EmptyState from '../EmptyState.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('EmptyState', () => {
  it('renders the legacy message prop as its description', () => {
    const wrapper = mount(EmptyState, {
      props: { title: 'Nothing here', message: 'Try a different filter' },
      global: { stubs: { Icon: true } },
    })
    expect(wrapper.find('.empty-state-description').text()).toBe('Try a different filter')
    wrapper.unmount()
  })
})
