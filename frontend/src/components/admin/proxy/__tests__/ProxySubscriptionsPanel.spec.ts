import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import type { ProxySubscriptionSource } from '@/types'
import ProxySubscriptionsPanel from '../ProxySubscriptionsPanel.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => (params ? `${key}:${JSON.stringify(params)}` : key)
    })
  }
})

function buildItem(overrides: Partial<ProxySubscriptionSource> = {}): ProxySubscriptionSource {
  return {
    id: 1,
    name: 'Primary Subscription',
    url: 'https://example.com/subscription.yaml',
    source_format: 'clash_yaml',
    enabled: true,
    refresh_interval_hours: 6,
    target_entry_count: 3,
    auto_add_to_pool: true,
    last_refreshed_at: '2026-05-19T12:00:00Z',
    last_success_at: '2026-05-19T12:00:00Z',
    last_error: '',
    last_node_count: 42,
    last_materialized_proxy_count: 9,
    created_at: '2026-05-19T11:00:00Z',
    updated_at: '2026-05-19T12:00:00Z',
    ...overrides
  }
}

function findButton(wrapper: ReturnType<typeof mount>, text: string) {
  return wrapper.findAll('button').find((button) => button.text().includes(text))
}

describe('ProxySubscriptionsPanel', () => {
  it('renders loading and empty states', async () => {
    const wrapper = mount(ProxySubscriptionsPanel, {
      props: {
        loading: true,
        items: []
      }
    })

    expect(wrapper.text()).toContain('common.loading')

    await wrapper.setProps({
      loading: false
    })

    expect(wrapper.text()).toContain('admin.proxies.subscriptions.empty')
  })

  it('renders subscription details and emits row actions', async () => {
    const item = buildItem({
      last_error: 'refresh failed'
    })

    const wrapper = mount(ProxySubscriptionsPanel, {
      props: {
        loading: false,
        items: [item]
      }
    })

    expect(wrapper.text()).toContain('Primary Subscription')
    expect(wrapper.text()).toContain('https://example.com/subscription.yaml')
    expect(wrapper.text()).toContain('clash_yaml')
    expect(wrapper.text()).toContain('refresh failed')
    expect(wrapper.text()).toContain('common.enabled')

    const refreshButton = findButton(wrapper, 'admin.proxies.subscriptions.refreshNow')
    const editButton = findButton(wrapper, 'common.edit')
    const viewNodesButton = findButton(wrapper, 'admin.proxies.subscriptions.viewNodes')
    const deleteButton = findButton(wrapper, 'common.delete')

    expect(refreshButton).toBeDefined()
    expect(editButton).toBeDefined()
    expect(viewNodesButton).toBeDefined()
    expect(deleteButton).toBeDefined()

    await refreshButton!.trigger('click')
    await editButton!.trigger('click')
    await viewNodesButton!.trigger('click')
    await deleteButton!.trigger('click')

    expect(wrapper.emitted('refresh')?.[0]).toEqual([1])
    expect(wrapper.emitted('edit')?.[0]).toEqual([item])
    expect(wrapper.emitted('view-nodes')?.[0]).toEqual([1])
    expect(wrapper.emitted('delete')?.[0]).toEqual([1])
  })
})
