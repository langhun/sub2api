<template>
  <section>
    <div class="mb-4 flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
      <div>
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ title }}
        </h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
          {{ subtitle }}
        </p>
      </div>
      <div class="grid grid-cols-2 gap-2 sm:min-w-[220px]">
        <div class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-800/80">
          <div class="text-[11px] text-gray-500 dark:text-dark-400">
            {{ valueLabel }}
          </div>
          <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
            {{ formatValue(totalValue) }}
          </div>
        </div>
        <div class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-800/80">
          <div class="text-[11px] text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.totalUsers') }}
          </div>
          <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
            {{ totalUsers }}
          </div>
        </div>
      </div>
    </div>

    <div
      v-if="displayEntries.length > 0 && chartData"
      data-testid="leaderboard-chart-layout"
      class="flex flex-col gap-6 xl:flex-row xl:items-center xl:gap-8"
    >
      <div
        data-testid="leaderboard-chart-wrapper"
        class="mx-auto w-full max-w-[16rem] xl:mx-0 xl:max-w-[24rem] xl:flex-[0_0_24rem]"
      >
        <div class="mx-auto h-56 w-56 sm:h-60 sm:w-60 xl:h-64 xl:w-64">
          <Doughnut :data="chartData" :options="doughnutOptions" />
        </div>
        <p class="mt-3 text-center text-xs text-gray-400 dark:text-dark-500">
          {{ hoverHint }}
        </p>
      </div>

      <div
        data-testid="leaderboard-chart-ranking-scroll"
        class="leaderboard-chart-ranking-scroll min-w-0 max-h-[34rem] flex-1 overflow-y-auto pr-1"
      >
        <table class="w-full text-xs">
          <thead>
            <tr class="text-gray-500 dark:text-gray-400">
              <th class="pb-2 text-left">{{ t('leaderboard.title') }}</th>
              <th class="pb-2 text-right">{{ metricLabel }}</th>
              <th class="pb-2 text-right">{{ valueLabel }}</th>
              <th class="pb-2 text-right">{{ t('leaderboard.share') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="entry in displayEntries"
              :key="`${entry.rank}-${entry.username}`"
              data-testid="leaderboard-chart-ranking-row"
              class="border-t border-gray-100 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-dark-700/40"
            >
              <td class="py-2">
                <div class="flex min-w-0 items-center gap-3">
                  <div
                    :class="rankClass(entry.rank)"
                    class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-sm font-bold"
                  >
                    <span v-if="entry.rank <= 3">{{ medals[entry.rank - 1] }}</span>
                    <span v-else class="text-gray-500 dark:text-dark-400">{{ entry.rank }}</span>
                  </div>
                  <div class="min-w-0">
                    <div class="flex items-center gap-2">
                      <span
                        class="h-2.5 w-2.5 shrink-0 rounded-full"
                        :style="{ backgroundColor: getEntryColor(entry.rank) }"
                      ></span>
                      <span
                        class="block max-w-[180px] truncate font-medium text-gray-900 dark:text-white"
                        :title="entry.username"
                      >
                        {{ entry.username }}
                      </span>
                    </div>
                    <div
                      v-if="formatSubtitle(entry)"
                      class="mt-1 text-[11px] text-gray-400 dark:text-dark-500"
                    >
                      {{ formatSubtitle(entry) }}
                    </div>
                  </div>
                </div>
              </td>
              <td class="py-2 text-right text-gray-600 dark:text-gray-400">
                {{ formatMetric(entry) }}
              </td>
              <td class="py-2 text-right text-green-600 dark:text-green-400">
                {{ formatValue(entry.value) }}
              </td>
              <td class="py-2 text-right text-gray-400 dark:text-gray-500">
                {{ formatShare(entry.value) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div
      v-else
      class="flex h-40 items-center justify-center text-sm text-gray-400 dark:text-dark-500"
    >
      {{ t('leaderboard.empty') }}
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArcElement, Chart as ChartJS, Legend, Tooltip } from 'chart.js'
import { Doughnut } from 'vue-chartjs'

import type { LeaderboardChartItem, LeaderboardEntry, LeaderboardSummary } from '@/api/leaderboard'
import { createConsumptionLeaderboardPalette } from './consumptionChartPalette'

ChartJS.register(ArcElement, Tooltip, Legend)

type ValueType = 'currency' | 'number'
type SubtitleType = 'balance' | 'consumption' | 'checkin'

const props = withDefaults(defineProps<{
  chartItems: LeaderboardChartItem[]
  title: string
  subtitle: string
  summary?: LeaderboardSummary | null
  entries?: LeaderboardEntry[]
  valueLabel?: string
  metricLabel?: string
  hoverHint?: string
  valueType?: ValueType
  subtitleType?: SubtitleType
}>(), {
  summary: null,
  entries: () => [],
  valueLabel: undefined,
  metricLabel: undefined,
  hoverHint: undefined,
  valueType: 'currency',
  subtitleType: 'consumption',
})

const { t } = useI18n()
const medals = ['??', '??', '??']

const valueLabel = computed(() => props.valueLabel ?? t('leaderboard.amount'))
const metricLabel = computed(() => props.metricLabel ?? t('leaderboard.requests'))
const hoverHint = computed(() => props.hoverHint ?? t('leaderboard.hoverHint'))
const chartColors = computed(() => createConsumptionLeaderboardPalette(props.chartItems.length))

const totalValue = computed(() => {
  if (typeof props.summary?.total_value === 'number') {
    return props.summary.total_value
  }
  return props.chartItems.reduce((sum, item) => sum + item.value, 0)
})

const totalUsers = computed(() => {
  if (typeof props.summary?.total_users === 'number') {
    return props.summary.total_users
  }
  return props.chartItems.length
})

const displayEntries = computed(() => {
  if (props.entries.length > 0) {
    return props.entries
  }
  return props.chartItems.map((item, index) => ({
    rank: index + 1,
    username: item.username,
    value: item.value,
    extra_int: undefined,
  }))
})

const chartData = computed(() => {
  if (!props.chartItems.length) {
    return null
  }

  return {
    labels: props.chartItems.map((item) => item.username),
    datasets: [
      {
        data: props.chartItems.map((item) => item.value),
        backgroundColor: chartColors.value,
        borderWidth: 0,
      },
    ],
  }
})

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false,
    },
    tooltip: {
      callbacks: {
        label: (context: any) => {
          const value = context.raw as number
          const total = context.dataset.data.reduce((a: number, b: number) => a + b, 0)
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          return `${context.label}: ${formatValue(value)} (${percentage}%)`
        },
      },
    },
  },
}))

function rankClass(rank: number): string {
  if (rank === 1) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (rank === 2) return 'bg-slate-200 text-slate-700 dark:bg-slate-700/70 dark:text-slate-200'
  if (rank === 3) return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300'
}

function getEntryColor(rank: number): string {
  return chartColors.value[Math.max(rank - 1, 0)] || 'hsl(215 16% 56%)'
}

function formatShare(value: number): string {
  const total = totalValue.value
  if (total <= 0) {
    return '0.0%'
  }
  return `${((value / total) * 100).toFixed(1)}%`
}

function formatMetric(entry: LeaderboardEntry): string {
  return typeof entry.extra_int === 'number' && entry.extra_int > 0 ? entry.extra_int.toLocaleString() : '-'
}

function formatSubtitle(entry: LeaderboardEntry): string {
  if (props.subtitleType === 'balance' && entry.extra_int) {
    return t('leaderboard.balanceSubtitle', { count: entry.extra_int })
  }
  if (props.subtitleType === 'checkin' && (entry.extra_int || entry.extra_date)) {
    return t('leaderboard.checkinSubtitle', {
      total: entry.extra_int || 0,
      date: entry.extra_date || '',
      reward: entry.extra_float?.toFixed(2) || '0.00',
    })
  }
  if (props.subtitleType === 'consumption' && entry.extra_int) {
    return t('leaderboard.consumptionSubtitle', { count: entry.extra_int })
  }
  return ''
}

function formatValue(value: number): string {
  if (props.valueType === 'number') {
    return value.toLocaleString()
  }
  return `$${formatCurrency(value)}`
}

function formatCurrency(value: number): string {
  if (value >= 1000) {
    return `${(value / 1000).toFixed(2)}K`
  }
  if (value >= 1) {
    return value.toFixed(2)
  }
  if (value >= 0.01) {
    return value.toFixed(3)
  }
  return value.toFixed(4)
}
</script>

<style scoped>
.leaderboard-chart-ranking-scroll {
  scrollbar-gutter: stable;
  scrollbar-width: thin;
  scrollbar-color: rgba(156, 163, 175, 0.6) transparent;
}

.leaderboard-chart-ranking-scroll::-webkit-scrollbar {
  width: 10px;
}

.leaderboard-chart-ranking-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.leaderboard-chart-ranking-scroll::-webkit-scrollbar-thumb {
  border-radius: 9999px;
  background: rgba(156, 163, 175, 0.55);
}

.leaderboard-chart-ranking-scroll::-webkit-scrollbar-thumb:hover {
  background: rgba(107, 114, 128, 0.8);
}

:global(.dark) .leaderboard-chart-ranking-scroll {
  scrollbar-color: rgba(75, 85, 99, 0.8) transparent;
}

:global(.dark) .leaderboard-chart-ranking-scroll::-webkit-scrollbar-thumb {
  background: rgba(75, 85, 99, 0.75);
}

:global(.dark) .leaderboard-chart-ranking-scroll::-webkit-scrollbar-thumb:hover {
  background: rgba(107, 114, 128, 0.85);
}
</style>