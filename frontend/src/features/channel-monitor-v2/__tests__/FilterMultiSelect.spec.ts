import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        key === 'channelMonitorV2.filters.labelValue'
          ? `${params?.label}: ${params?.value}`
          : key,
      locale: { value: 'en' },
    }),
  }
})

import FilterMultiSelect from '../FilterMultiSelect.vue'

describe('FilterMultiSelect', () => {
  it('keeps a teleported menu visible when the trigger is at the mobile right edge', async () => {
    Object.defineProperty(window, 'innerWidth', { configurable: true, value: 390 })
    const wrapper = mount(FilterMultiSelect, {
      props: {
        label: 'Platform',
        allLabel: 'All platforms',
        modelValue: [],
        options: [{ value: 'openai', label: 'OpenAI' }],
        compact: true,
      },
      global: { stubs: { Icon: true } },
    })
    const trigger = wrapper.get('button[aria-haspopup="listbox"]')
    vi.spyOn(trigger.element, 'getBoundingClientRect').mockReturnValue({
      left: 336,
      right: 390,
      top: 10,
      bottom: 42,
      width: 54,
      height: 32,
      x: 336,
      y: 10,
      toJSON: () => ({}),
    })

    await trigger.trigger('click')
    await nextTick()

    const dropdown = document.body.querySelector<HTMLElement>('.filter-dropdown')
    expect(dropdown).not.toBeNull()
    const left = Number.parseFloat(dropdown?.style.left || '0')
    const width = Number.parseFloat(dropdown?.style.minWidth || '0')
    expect(width).toBeGreaterThan(0)
    expect(left).toBeGreaterThanOrEqual(8)
    expect(left + width).toBeLessThanOrEqual(382)

    wrapper.unmount()
    dropdown?.remove()
  })
})
