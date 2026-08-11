import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import LocaleFlagIcon from '../LocaleFlagIcon.vue'

describe('LocaleFlagIcon', () => {
  it.each([
    ['zh', 'cn.svg'],
    ['en', 'us.svg'],
  ])('uses a bundled image for %s instead of a system emoji', (locale, assetName) => {
    const wrapper = mount(LocaleFlagIcon, { props: { locale } })
    const image = wrapper.get('img')

    expect(image.attributes('src')).toContain(assetName)
    expect(image.attributes('width')).toBe('20')
    expect(image.attributes('height')).toBe('15')
    expect(image.attributes('aria-hidden')).toBe('true')
    expect(wrapper.text()).toBe('')
  })

  it('does not render a misleading flag for an unknown locale', () => {
    const wrapper = mount(LocaleFlagIcon, { props: { locale: 'unknown' } })

    expect(wrapper.find('img').exists()).toBe(false)
  })
})
