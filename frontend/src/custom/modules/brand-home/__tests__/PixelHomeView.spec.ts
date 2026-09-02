import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const stores = vi.hoisted(() => ({
  app: { siteName: 'DaiGua' },
  auth: { isAuthenticated: false, isAdmin: false },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => stores.app,
  useAuthStore: () => stores.auth,
}))

import PixelHomeView from '../PixelHomeView.vue'

beforeEach(() => {
  stores.app.siteName = 'DaiGua'
  stores.auth.isAuthenticated = false
  stores.auth.isAdmin = false
})

describe('PixelHomeView', () => {
  it('keeps the supplied visual style while rendering only local branding and routes', () => {
    const wrapper = mount(PixelHomeView)

    expect(wrapper.text()).toContain('DaiGua')
    expect(wrapper.text()).not.toMatch(/Aivoza|Wei-Shaw|GitHub/i)
    expect(wrapper.html()).not.toMatch(/aivoza\.com|github\.com/i)
    expect(wrapper.get('.av-code').text()).toContain(`${window.location.origin}/v1/chat/completions`)
    expect(wrapper.get('a[href="/docs"]').text()).toBe('开发文档')
    expect(wrapper.get('a[href="/login"]').text()).toContain('进入控制台')
  })

  it('does not expose the upstream default name when public settings are unavailable', () => {
    stores.app.siteName = 'Sub2API'

    const wrapper = mount(PixelHomeView)

    expect(wrapper.text()).toContain('DaiGua')
    expect(wrapper.text()).not.toContain('Sub2API')
    expect(wrapper.text()).toContain('/v1/chat/completions')
  })

  it('sends authenticated members to their applicable console', () => {
    stores.auth.isAuthenticated = true
    stores.auth.isAdmin = true

    const wrapper = mount(PixelHomeView)

    expect(wrapper.get('a[href="/admin/dashboard"]').text()).toContain('进入控制台')
  })
})
