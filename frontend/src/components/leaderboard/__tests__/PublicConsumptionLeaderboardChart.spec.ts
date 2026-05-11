import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import PublicConsumptionLeaderboardChart from '../PublicConsumptionLeaderboardChart.vue'

const messages: Record<string, string> = {
  'leaderboard.title': '排行榜',
  'leaderboard.tabs.consumption': '消耗排行',
  'leaderboard.consumptionChartTitle': '消费分布',
  'leaderboard.consumptionChartSubtitle': '查看当前周期所有消费用户的金额占比',
  'leaderboard.consumptionSubtitle': '{count} 次请求',
  'leaderboard.totalAmount': '总金额',
  'leaderboard.totalUsers': '用户数',
  'leaderboard.hoverHint': '悬停圆环切片可查看用户、金额和占比',
  'leaderboard.requests': '请求',
  'leaderboard.amount': '金额',
  'leaderboard.share': '占比',
  'leaderboard.empty': '暂无数据',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string | number>) => {
        const template = messages[key] ?? key
        if (!params) return template
        return Object.entries(params).reduce(
          (result, [paramKey, value]) => result.replace(`{${paramKey}}`, String(value)),
          template,
        )
      },
    }),
  }
})

vi.mock('vue-chartjs', () => ({
  Doughnut: {
    props: ['data'],
    template: '<div class="chart-data">{{ JSON.stringify(data) }}</div>',
  },
}))

describe('PublicConsumptionLeaderboardChart', () => {
  it('uses all chart_items without merging others', () => {
    const chartItems = Array.from({ length: 25 }, (_, index) => ({
      username: `用户 ${index + 1}`,
      value: 25 - index,
    }))

    const wrapper = mount(PublicConsumptionLeaderboardChart, {
      props: {
        chartItems,
        summary: {
          total_value: chartItems.reduce((sum, item) => sum + item.value, 0),
          total_users: chartItems.length,
        },
      },
    })

    const chartData = JSON.parse(wrapper.find('.chart-data').text())
    expect(chartData.labels).toHaveLength(25)
    expect(chartData.labels[0]).toBe('用户 1')
    expect(chartData.labels[24]).toBe('用户 25')
    expect(chartData.labels).not.toContain('其他')
    expect(chartData.datasets[0].data).toHaveLength(25)
    expect(chartData.datasets[0].backgroundColor).toHaveLength(25)
    expect(chartData.datasets[0].backgroundColor.slice(0, 3)).toEqual(['#3b82f6', '#10b981', '#f59e0b'])
    expect(wrapper.text()).toContain('总金额')
    expect(wrapper.text()).toContain('用户数')
  })

  it('formats tooltip as amount plus percentage', () => {
    const wrapper = mount(PublicConsumptionLeaderboardChart, {
      props: {
        chartItems: [
          { username: 'Alpha', value: 60 },
          { username: 'Beta', value: 40 },
        ],
        summary: {
          total_value: 100,
          total_users: 2,
        },
      },
    })

    const options = (wrapper.vm as any).$?.setupState.doughnutOptions
    const label = options.plugins.tooltip.callbacks.label({
      label: 'Alpha',
      raw: 60,
    })

    expect(label).toBe('Alpha: $60.00 (60.0%)')
  })

  it('renders ranking rows beside the chart with share text', () => {
    const wrapper = mount(PublicConsumptionLeaderboardChart, {
      props: {
        chartItems: [
          { username: 'Alpha', value: 60 },
          { username: 'Beta', value: 40 },
        ],
        entries: [
          { rank: 1, username: 'Alpha', value: 60, extra_int: 12 },
          { rank: 2, username: 'Beta', value: 40, extra_int: 8 },
        ],
        summary: {
          total_value: 100,
          total_users: 2,
        },
      },
    })

    expect(wrapper.text()).toContain('Alpha')
    expect(wrapper.text()).toContain('60.0%')
    expect(wrapper.text()).toContain('12 次请求')
  })

  it('defaults to a 9-row viewport but keeps all ranking rows scrollable', () => {
    const entries = Array.from({ length: 12 }, (_, index) => ({
      rank: index + 1,
      username: `用户${index + 1}`,
      value: 120 - index,
      extra_int: 10 + index,
    }))

    const wrapper = mount(PublicConsumptionLeaderboardChart, {
      props: {
        chartItems: entries.map(({ username, value }) => ({ username, value })),
        entries,
        summary: {
          total_value: entries.reduce((sum, entry) => sum + entry.value, 0),
          total_users: entries.length,
        },
      },
    })

    const scrollContainer = wrapper.get('[data-testid="consumption-ranking-scroll"]')
    const rankingRows = wrapper.findAll('[data-testid=\"consumption-ranking-row\"]')
    expect(scrollContainer.classes()).toContain('consumption-ranking-scroll')
    expect(scrollContainer.classes()).toContain('max-h-[24rem]')
    expect(scrollContainer.classes()).toContain('overflow-y-auto')
    expect(rankingRows).toHaveLength(12)
    expect(rankingRows[8].text()).toContain('用户9')
    expect(rankingRows[9].text()).toContain('用户10')
    expect(rankingRows[11].text()).toContain('用户12')
  })
})
