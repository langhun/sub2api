import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

const { appStore, replace } = vi.hoisted(() => ({
  appStore: {
    publicSettingsLoaded: false,
    cachedPublicSettings: null as { default_homepage: 'default' | 'dino' } | null,
    fetchPublicSettings: vi.fn(),
  },
  replace: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({ replace }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
}))

import RootHomeView from '../RootHomeView.vue'

describe('brand-home root redirect', () => {
  it.each([
    ['default', '/home'],
    ['dino', '/Dino'],
  ] as const)('loads %s default_homepage before redirecting to %s', async (defaultHomepage, target) => {
    appStore.publicSettingsLoaded = false
    appStore.cachedPublicSettings = null
    appStore.fetchPublicSettings.mockReset()
    appStore.fetchPublicSettings.mockImplementation(async () => {
      appStore.cachedPublicSettings = { default_homepage: defaultHomepage }
      appStore.publicSettingsLoaded = true
    })
    replace.mockReset()

    mount(RootHomeView)
    await flushPromises()

    expect(appStore.fetchPublicSettings).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith(target)
  })
})
