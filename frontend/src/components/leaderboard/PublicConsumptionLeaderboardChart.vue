<template>
  <section>
    <div class="mb-4 flex flex-col gap-3 xl:flex-row xl:items-start xl:justify-between">
      <div>
        <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('leaderboard.consumptionChartTitle') }}
        </h2>
        <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
          {{ t('leaderboard.consumptionChartSubtitle') }}
        </p>
      </div>
      <div class="grid grid-cols-2 gap-2 sm:min-w-[220px]">
        <div class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-800/80">
          <div class="text-[11px] text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.totalAmount') }}
          </div>
          <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
            ${{ formatCurrency(totalValue) }}
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

    <div v-if="displayEntries.length > 0 && chartData" class="flex flex-col gap-6 xl:flex-row xl:items-center">
      <div class="h-48 w-48 shrink-0">
        <Doughnut :data="chartData" :options="doughnutOptions" />
      </div>

      <div
        data-testid="consumption-ranking-scroll"
        class="consumption-ranking-scroll max-h-[24rem] flex-1 overflow-y-auto"
      >
        <table class="w-full text-xs">
          <thead>
            <tr class="text-gray-500 dark:text-gray-400">
              <th class="pb-2 text-left">{{ t('leaderboard.title') }}</th>
              <th class="pb-2 text-right">{{ t('leaderboard.requests') }}</th>
              <th class="pb-2 text-right">{{ t('leaderboard.amount') }}</th>
              <th class="pb-2 text-right">{{ t('leaderboard.share') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(entry, index) in displayEntries"
              :key="`${entry.rank}-${entry.username}`"
              data-testid="consumption-ranking-row"
              class="border-t border-gray-100 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-dark-700/40"
            >
              <td class="py-1.5">
                <div class="flex min-w-0 items-center gap-2">
                  <span class="shrink-0 text-[11px] font-semibold text-gray-500 dark:text-gray-400">
                    #{{ resolveRank(entry.rank, index) }}
                  </span>
                  <span
                    class="block max-w-[140px] truncate font-medium text-gray-900 dark:text-white"
                    :title="entry.username"
                  >
                    {{ entry.username }}
                  </span>
                </div>
              </td>
              <td class="py-1.5 text-right text-gray-600 dark:text-gray-400">
                {{ formatRequestCount(entry.extra_int) }}
              </td>
              <td class="py-1.5 text-right text-green-600 dark:text-green-400">
                ${{ formatCurrency(entry.value) }}
              </td>
              <td class="py-1.5 text-right text-gray-400 dark:text-gray-500">
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

const props = withDefaults(defineProps<{
  chartItems: LeaderboardChartItem[]
  summary?: LeaderboardSummary | null
  entries?: LeaderboardEntry[]
}>(), {
  summary: null,
  entries: () => [],
})

const { t } = useI18n()

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
          return `${context.label}: $${formatCurrency(value)} (${percentage}%)`
        },
      },
    },
  },
}))

function resolveRank(rank: number | undefined, index: number): number {
  return typeof rank === 'number' && rank > 0 ? rank : index + 1
}

function formatShare(value: number): string {
  const total = totalValue.value
  if (total <= 0) {
    return '0.0%'
  }
  return `${((value / total) * 100).toFixed(1)}%`
}

function formatRequestCount(count?: number): string {
  return typeof count === 'number' && count > 0 ? count.toLocaleString() : '-'
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
.consumption-ranking-scroll {
  scrollbar-gutter: stable;
  scrollbar-width: thin;
  scrollbar-color: rgba(156, 163, 175, 0.6) transparent;
}

.consumption-ranking-scroll::-webkit-scrollbar {
  width: 10px;
}

.consumption-ranking-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.consumption-ranking-scroll::-webkit-scrollbar-thumb {
  border-radius: 9999px;
  background: rgba(156, 163, 175, 0.55);
}

.consumption-ranking-scroll::-webkit-scrollbar-thumb:hover {
  background: rgba(107, 114, 128, 0.8);
}

:global(.dark) .consumption-ranking-scroll {
  scrollbar-color: rgba(75, 85, 99, 0.8) transparent;
}

:global(.dark) .consumption-ranking-scroll::-webkit-scrollbar-thumb {
  background: rgba(75, 85, 99, 0.75);
}

:global(.dark) .consumption-ranking-scroll::-webkit-scrollbar-thumb:hover {
  background: rgba(107, 114, 128, 0.85);
}
</style>
