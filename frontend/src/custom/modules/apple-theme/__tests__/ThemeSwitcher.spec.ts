import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ThemeSwitcher from '../ThemeSwitcher.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => ({
      'theme.toggle': '切换主题',
      'theme.modeLabel': '明暗模式',
      'theme.modeLight': '浅色',
      'theme.modeDark': '深色',
      'theme.modeAuto': '跟随系统',
    })[key] ?? key,
  }),
}))

describe('ThemeSwitcher', () => {
  it('keeps the theme menu open after clicking the trigger', async () => {
    const wrapper = mount(ThemeSwitcher, {
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    await wrapper.get('button[aria-label="切换主题"]').trigger('click')

    expect(wrapper.find('.dropdown').exists()).toBe(true)
    expect(wrapper.text()).toContain('明暗模式')
    expect(wrapper.text()).not.toContain('Apple Glass')
  })
})
