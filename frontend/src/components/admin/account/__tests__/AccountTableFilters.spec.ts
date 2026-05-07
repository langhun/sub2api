import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import AccountTableFilters from '../AccountTableFilters.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const SelectStub = defineComponent({
  name: 'SelectStub',
  props: {
    options: {
      type: Array,
      default: () => []
    }
  },
  setup(props) {
    return () =>
      h(
        'div',
        { class: 'select-stub' },
        (props.options as Array<{ label?: string; value?: string }>)
          .map((option) => option.label || option.value || '')
          .join('|')
      )
  }
})

const SearchInputStub = defineComponent({
  name: 'SearchInputStub',
  setup() {
    return () => h('div', { class: 'search-input-stub' })
  }
})

describe('AccountTableFilters', () => {
  it('默认收起高级筛选并可展开查看完整状态筛选项', async () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          tier: '',
          type: '',
          main_status: '',
          runtime_status: '',
          scheduling_status: '',
          privacy_mode: '',
          group: ''
        },
        groups: []
      },
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub
        }
      }
    })

    const text = wrapper.text()

    expect(text).toContain('admin.accounts.allPlatforms')
    expect(text).toContain('admin.accounts.statusFilters.allMain')
    expect(text).toContain('admin.accounts.statusFilters.allRuntime')
    expect(text).toContain('admin.accounts.moreFilters')
    expect(text).not.toContain('admin.accounts.tier.all')
    expect(text).not.toContain('admin.accounts.statusFilters.allScheduling')

    await wrapper.get('[data-testid="account-more-filters-toggle"]').trigger('click')

    const expandedText = wrapper.text()

    expect(expandedText).toContain('admin.accounts.hideAdvancedFilters')
    expect(expandedText).toContain('admin.accounts.tier.all')
    expect(text).toContain('admin.accounts.status.mainActive')
    expect(text).toContain('admin.accounts.status.mainInactive')
    expect(text).toContain('admin.accounts.status.mainError')
    expect(expandedText).toContain('admin.accounts.status.runtimeNormal')
    expect(expandedText).toContain('admin.accounts.status.runtimeRateLimited')
    expect(expandedText).toContain('admin.accounts.status.runtimeOverloaded')
    expect(expandedText).toContain('admin.accounts.statusFilters.tempUnschedulable')
    expect(expandedText).toContain('admin.accounts.statusFilters.allScheduling')
    expect(expandedText).toContain('admin.accounts.status.scheduleEnabled')
    expect(expandedText).toContain('admin.accounts.statusFilters.unschedulable')
    expect(expandedText).toContain('admin.accounts.statusGuide.shortAction')
  })

  it('点击状态说明按钮会发出事件', async () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          tier: '',
          type: '',
          main_status: '',
          runtime_status: '',
          scheduling_status: '',
          privacy_mode: '',
          group: ''
        },
        groups: []
      },
      global: {
        stubs: {
          Select: SelectStub,
          SearchInput: SearchInputStub
        }
      }
    })

    await wrapper.get('[data-testid="account-status-guide-button"]').trigger('click')
    expect(wrapper.emitted('status-guide')).toHaveLength(1)
  })
})
