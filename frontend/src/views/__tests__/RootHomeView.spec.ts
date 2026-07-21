import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

const { appStore, replace } = vi.hoisted(() => ({
  appStore: {
    publicSettingsLoaded: true,
    cachedPublicSettings: { default_homepage: 'default' as 'default' | 'dino' },
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

describe('RootHomeView', () => {
  it.each([
    ['default', '/home'],
    ['dino', '/Dino'],
  ] as const)('redirects %s homepage mode to %s', async (defaultHomepage, target) => {
    appStore.publicSettingsLoaded = true
    appStore.cachedPublicSettings = { default_homepage: defaultHomepage }
    replace.mockReset()

    mount(RootHomeView)
    await flushPromises()

    expect(replace).toHaveBeenCalledWith(target)
  })

  it('falls back to the default page when public settings fail to load', async () => {
    appStore.publicSettingsLoaded = false
    appStore.cachedPublicSettings = null
    appStore.fetchPublicSettings.mockRejectedValueOnce(new Error('network failure'))
    replace.mockReset()

    mount(RootHomeView)
    await flushPromises()

    expect(replace).toHaveBeenCalledWith('/home')
  })
})
