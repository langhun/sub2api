import { describe, expect, it, vi, beforeEach } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import UsageFilters from '../UsageFilters.vue'

const { groupsList, getModelStats } = vi.hoisted(() => ({
  groupsList: vi.fn(),
  getModelStats: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      list: groupsList,
    },
    dashboard: {
      getModelStats,
    },
  },
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const SelectStub = {
  props: ['modelValue', 'options'],
  emits: ['change', 'update:modelValue'],
  template: '<button type="button" class="select-stub">{{ options.length }}</button>',
}

describe('admin usage filters', () => {
  beforeEach(() => {
    groupsList.mockReset()
    getModelStats.mockReset()
    groupsList.mockResolvedValue({ items: [] })
    getModelStats.mockResolvedValue({ models: [{ model: 'gpt-5.4' }] })
  })

  it('defers model stats option loading until the model filter is focused', async () => {
    const wrapper = mount(UsageFilters, {
      props: {
        modelValue: {},
        exporting: false,
        startDate: '2026-05-03',
        endDate: '2026-05-04',
      },
      global: {
        stubs: {
          Select: SelectStub,
        },
      },
    })

    await flushPromises()
    expect(groupsList).toHaveBeenCalledTimes(1)
    expect(getModelStats).not.toHaveBeenCalled()

    await wrapper.findAll('.select-stub')[0].trigger('focusin')
    await flushPromises()

    expect(getModelStats).toHaveBeenCalledTimes(1)
    expect(getModelStats).toHaveBeenCalledWith({
      start_date: '2026-05-03',
      end_date: '2026-05-04',
    })
  })
})
