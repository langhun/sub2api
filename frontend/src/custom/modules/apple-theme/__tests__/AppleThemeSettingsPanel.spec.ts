import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AppleThemeSettingsPanel from '../admin/AppleThemeSettingsPanel.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

describe('AppleThemeSettingsPanel', () => {
  it('uses the custom settings model contract and emits changes', async () => {
    const wrapper = mount(AppleThemeSettingsPanel, {
      props: {
        modelValue: {
          enable_apple_theme: false,
          default_theme_mode: 'auto',
        },
      },
    })

    await wrapper.find('button.switch').trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({
      enable_apple_theme: true,
      default_theme_mode: 'auto',
    })
    expect(wrapper.find('select').exists()).toBe(true)
  })
})
