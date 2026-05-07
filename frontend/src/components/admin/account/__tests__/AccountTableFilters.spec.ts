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
  it('状态筛选项使用更明确的分层文案', () => {
    const wrapper = mount(AccountTableFilters, {
      props: {
        searchQuery: '',
        filters: {
          platform: '',
          tier: '',
          type: '',
          status: '',
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

    expect(text).toContain('admin.accounts.statusFilters.active')
    expect(text).toContain('admin.accounts.statusFilters.inactive')
    expect(text).toContain('admin.accounts.statusFilters.error')
    expect(text).toContain('admin.accounts.statusFilters.rateLimited')
    expect(text).toContain('admin.accounts.statusFilters.tempUnschedulable')
    expect(text).toContain('admin.accounts.statusFilters.unschedulable')
  })
})
