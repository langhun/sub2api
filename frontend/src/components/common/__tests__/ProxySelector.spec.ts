import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import ProxySelector from '../ProxySelector.vue'
import type { Proxy } from '@/types'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string) => {
      const messages: Record<string, string> = {
        'admin.accounts.noProxy': 'No proxy',
        'admin.proxies.searchProxies': 'Search proxies',
        'admin.proxies.batchTest': 'Batch test',
        'admin.proxies.testConnection': 'Test connection',
        'admin.proxies.testFailed': 'Test failed',
        'common.noOptionsFound': 'No options found',
      }
      return messages[key] ?? key
    },
  }),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    proxies: {
      testProxy: vi.fn(),
    },
  },
}))

const proxies = [
  {
    id: 1,
    name: 'Primary proxy',
    protocol: 'http',
    host: '127.0.0.1',
    port: 7890,
    account_count: 3,
  },
] as Proxy[]

describe('ProxySelector', () => {
  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('teleports the dropdown to body with fixed positioning to avoid modal clipping', async () => {
    const wrapper = mount(ProxySelector, {
      props: {
        modelValue: null,
        proxies,
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    await wrapper.find('.select-trigger').trigger('click')
    await nextTick()

    const dropdown = document.body.querySelector<HTMLElement>('.select-dropdown')
    expect(wrapper.find('.select-dropdown').exists()).toBe(false)
    expect(dropdown).not.toBeNull()
    expect(dropdown?.style.position).toBe('fixed')
    expect(dropdown?.style.zIndex).toBe('100000020')
  })
})
