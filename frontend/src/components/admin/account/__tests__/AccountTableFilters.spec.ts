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
  it('默认直接展示全部筛选项，不再使用更多筛选折叠', async () => {
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
    expect(text).toContain('admin.accounts.allGroups')
    expect(text).toContain('admin.accounts.tier.all')
    expect(text).toContain('admin.accounts.status.mainActive')
    expect(text).toContain('admin.accounts.status.mainInactive')
    expect(text).toContain('admin.accounts.status.mainError')
    expect(text).toContain('admin.accounts.status.runtimeNormal')
    expect(text).toContain('admin.accounts.status.runtimeRateLimited')
    expect(text).toContain('admin.accounts.status.runtimeOverloaded')
    expect(text).toContain('admin.accounts.statusFilters.tempUnschedulable')
    expect(text).toContain('admin.accounts.statusFilters.allScheduling')
    expect(text).toContain('admin.accounts.status.scheduleEnabled')
    expect(text).toContain('admin.accounts.statusFilters.unschedulable')
    expect(text).not.toContain('admin.accounts.moreFilters')
    expect(wrapper.find('[data-testid="account-more-filters-toggle"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="account-status-guide-button"]').exists()).toBe(false)
  })

  it('不再渲染筛选区状态说明按钮', async () => {
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

    expect(wrapper.find('[data-testid="account-status-guide-button"]').exists()).toBe(false)
  })
})
