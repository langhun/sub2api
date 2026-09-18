import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AppleThemeSettingsPanel from '../admin/AppleThemeSettingsPanel.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key }),
}))

describe('AppleThemeSettingsPanel', () => {
  it('defaults the global theme to enabled and automatic mode', async () => {
    const { appleThemeSettingsPanels } = await import('../settings')
    expect(appleThemeSettingsPanels[0]?.createForm()).toEqual({
      enable_apple_theme: true,
      default_theme_mode: 'auto',
    })
  })

  it('uses the custom settings model contract and emits changes', async () => {
    const wrapper = mount(AppleThemeSettingsPanel, {
      props: {
        modelValue: {
          enable_apple_theme: false,
          default_theme_mode: 'auto',
        },
      },
    })

    const toggle = wrapper.get('[role="switch"]')
    expect(toggle.attributes('aria-checked')).toBe('false')
    expect(toggle.attributes('aria-label')).toBe('custom.appleTheme.enableLabel')
    await toggle.trigger('click')
    expect(toggle.attributes('aria-checked')).toBe('true')

    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({
      enable_apple_theme: true,
      default_theme_mode: 'auto',
    })
    expect(wrapper.find('select').exists()).toBe(true)
  })
})
