import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UserDashboardCharts from '../UserDashboardCharts.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('vue-chartjs', () => ({
  Doughnut: { template: '<div />' }
}))

describe('UserDashboardCharts', () => {
  it('keeps the date menu container above the chart cards', () => {
    const wrapper = mount(UserDashboardCharts, {
      props: {
        loading: false,
        startDate: '2026-09-20',
        endDate: '2026-09-21',
        granularity: 'hour',
        trend: [],
        models: []
      },
      global: {
        stubs: {
          DateRangePicker: true,
          Select: true,
          TokenUsageTrend: true
        }
      }
    })

    expect(wrapper.find('.card.relative.z-20.overflow-visible').exists()).toBe(true)
  })
})
