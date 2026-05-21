import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import ProxiesToolbar from '../ProxiesToolbar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const SelectStub = {
  props: ['modelValue', 'options', 'placeholder'],
  emits: ['update:modelValue', 'change'],
  template: `
    <button
      type="button"
      data-test="select-stub"
      :data-placeholder="placeholder"
      @click="$emit('update:modelValue', 'changed'); $emit('change')"
    >
      {{ placeholder }}
    </button>
  `
}

const baseProps = {
  activeTab: 'proxies' as const,
  searchQuery: '',
  filters: {
    protocol: '',
    status: '',
    runtime_status: ''
  },
  protocolOptions: [{ value: '', label: 'all' }],
  statusOptions: [{ value: '', label: 'all' }],
  runtimeStatusOptions: [{ value: '', label: 'all' }],
  loading: false,
  loadingSubscriptions: false,
  batchTesting: false,
  batchQualityChecking: false,
  selectedCount: 0,
  showColumnDropdown: false,
  showProxyToolsDropdown: false,
  showProxyBatchDropdown: false,
  toggleableColumns: [{ key: 'auth', label: 'auth' }],
  isColumnVisible: () => true
}

describe('ProxiesToolbar', () => {
  it('emits tab switch and search updates', async () => {
    const wrapper = mount(ProxiesToolbar, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          Icon: true
        }
      }
    })

    const buttons = wrapper.findAll('button')
    const subscriptionsTab = buttons.find((button) => button.text().includes('admin.proxies.subscriptions.tab'))
    expect(subscriptionsTab).toBeDefined()
    await subscriptionsTab!.trigger('click')

    const search = wrapper.get('input')
    await search.setValue('proxy-host')

    expect(wrapper.emitted('set-tab')?.[0]).toEqual(['subscriptions'])
    expect(wrapper.emitted('update:searchQuery')?.[0]).toEqual(['proxy-host'])
  })

  it('emits filter updates and reload for proxy filters', async () => {
    const wrapper = mount(ProxiesToolbar, {
      props: baseProps,
      global: {
        stubs: {
          Select: SelectStub,
          Icon: true
        }
      }
    })

    const selects = wrapper.findAll('[data-test="select-stub"]')
    expect(selects).toHaveLength(3)

    await selects[0].trigger('click')
    await selects[1].trigger('click')
    await selects[2].trigger('click')

    const updates = wrapper.emitted('update:filters') || []
    expect(updates).toHaveLength(3)
    expect(updates[0]?.[0]).toMatchObject({ protocol: 'changed', status: '', runtime_status: '' })
    expect(updates[1]?.[0]).toMatchObject({ protocol: '', status: 'changed', runtime_status: '' })
    expect(updates[2]?.[0]).toMatchObject({ protocol: '', status: '', runtime_status: 'changed' })
    expect(wrapper.emitted('reload-proxies')).toHaveLength(3)
  })

  it('emits tools and batch actions from dropdown menus', async () => {
    const wrapper = mount(ProxiesToolbar, {
      props: {
        ...baseProps,
        selectedCount: 2,
        showProxyBatchDropdown: true,
        showColumnDropdown: true
      },
      global: {
        stubs: {
          Select: SelectStub,
          Icon: true
        }
      }
    })

    const columnItem = wrapper.findAll('button').find((button) => button.text().includes('auth'))
    expect(columnItem).toBeDefined()
    await columnItem!.trigger('click')

    const textButtons = wrapper.findAll('button')
    const toolsImport = wrapper.get('[data-test="proxy-toolbar-import"]')
    const toolsPool = wrapper.get('[data-test="proxy-toolbar-pool"]')
    const batchDelete = wrapper.get('[data-test="proxy-toolbar-batch-delete"]')
    const batchAssign = wrapper.get('[data-test="proxy-toolbar-batch-assign"]')

    expect(textButtons.some((button) => button.text().includes('admin.proxies.dataExportSelected'))).toBe(true)

    await toolsImport.trigger('click')
    await toolsPool.trigger('click')
    await batchAssign.trigger('click')
    await batchDelete.trigger('click')

    expect(wrapper.emitted('toggle-column')?.[0]).toEqual(['auth'])
    expect(wrapper.emitted('open-import')).toHaveLength(1)
    expect(wrapper.emitted('open-pool')).toHaveLength(1)
    expect(wrapper.emitted('batch-assign')).toHaveLength(1)
    expect(wrapper.emitted('batch-delete')).toHaveLength(1)
  })

  it('renders subscription-specific actions on subscriptions tab', async () => {
    const wrapper = mount(ProxiesToolbar, {
      props: {
        ...baseProps,
        activeTab: 'subscriptions'
      },
      global: {
        stubs: {
          Select: SelectStub,
          Icon: true
        }
      }
    })

    expect(wrapper.find('input').exists()).toBe(false)
    const buttons = wrapper.findAll('button')
    const refresh = buttons.find((button) => button.attributes('title') !== 'admin.proxies.subscriptions.create')
    const create = buttons.find((button) => button.text().includes('admin.proxies.subscriptions.create'))
    expect(refresh).toBeDefined()
    expect(create).toBeDefined()

    await create!.trigger('click')
    expect(wrapper.emitted('create-subscription')).toHaveLength(1)
  })
})
