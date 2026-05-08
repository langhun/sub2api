<template>
  <section
    class="overflow-hidden rounded-[28px] border border-gray-100 bg-gradient-to-br from-white via-white to-primary-50/40 dark:border-dark-700/60 dark:from-dark-900 dark:via-dark-900 dark:to-primary-950/20"
  >
    <div class="border-b border-gray-100/80 px-4 py-4 dark:border-dark-700/60 sm:px-6">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div>
          <h2 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('leaderboard.consumptionChartTitle') }}
          </h2>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.consumptionChartSubtitle') }}
          </p>
        </div>
        <div class="grid grid-cols-2 gap-2 sm:min-w-[220px]">
          <div
            class="rounded-2xl border border-gray-100 bg-white/85 px-3 py-2.5 shadow-sm shadow-primary-100/30 dark:border-dark-700/60 dark:bg-dark-900/70 dark:shadow-none"
          >
            <div class="text-[11px] text-gray-500 dark:text-dark-400">
              {{ t('leaderboard.totalAmount') }}
            </div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
              ${{ formatCurrency(totalValue) }}
            </div>
          </div>
          <div
            class="rounded-2xl border border-gray-100 bg-white/85 px-3 py-2.5 shadow-sm shadow-primary-100/30 dark:border-dark-700/60 dark:bg-dark-900/70 dark:shadow-none"
          >
            <div class="text-[11px] text-gray-500 dark:text-dark-400">
              {{ t('leaderboard.totalUsers') }}
            </div>
            <div class="mt-1 text-sm font-semibold text-gray-900 dark:text-white">
              {{ totalUsers }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="grid gap-5 px-4 py-5 xl:grid-cols-[minmax(240px,300px)_minmax(0,1fr)] sm:px-6">
      <div
        class="flex flex-col items-center justify-center rounded-3xl border border-gray-100 bg-white/85 p-4 shadow-sm shadow-primary-100/40 dark:border-dark-700/60 dark:bg-dark-900/70 dark:shadow-none"
      >
        <div class="h-64 w-full max-w-[280px]">
          <Doughnut v-if="chartData" :data="chartData" :options="doughnutOptions" />
        </div>
        <p class="mt-4 text-center text-xs text-gray-400 dark:text-dark-500">
          {{ t('leaderboard.hoverHint') }}
        </p>
      </div>

      <div
        class="overflow-hidden rounded-3xl border border-gray-100 bg-white/85 shadow-sm shadow-primary-100/30 dark:border-dark-700/60 dark:bg-dark-900/70 dark:shadow-none"
      >
        <div class="border-b border-gray-100/80 px-4 py-4 dark:border-dark-700/60 sm:px-5">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ t('leaderboard.title') }}
          </p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.tabs.consumption') }}
          </p>
        </div>

        <div
          v-if="displayEntries.length > 0"
          data-testid="consumption-ranking-scroll"
          class="consumption-ranking-scroll max-h-[612px] overflow-y-auto px-2 py-2 pr-1 sm:px-3"
        >
          <div
            v-for="entry in displayEntries"
            :key="`${entry.rank}-${entry.username}`"
            data-testid="consumption-ranking-row"
            class="group flex items-center gap-3 rounded-2xl px-3 py-3 transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/50"
          >
            <div
              :class="rankClass(entry.rank)"
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-2xl text-sm font-bold"
            >
              <span v-if="entry.rank <= 3">{{ medals[entry.rank - 1] }}</span>
              <span v-else class="text-gray-500 dark:text-dark-400">{{ entry.rank }}</span>
            </div>

            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span
                  class="h-2.5 w-2.5 shrink-0 rounded-full"
                  :style="{ backgroundColor: getEntryColor(entry.rank) }"
                ></span>
                <p class="truncate text-sm font-medium text-gray-900 dark:text-white">
                  {{ entry.username }}
                </p>
              </div>
              <p
                v-if="entry.extra_int"
                class="mt-1 truncate text-xs text-gray-400 dark:text-dark-500"
              >
                {{ t('leaderboard.consumptionSubtitle', { count: entry.extra_int }) }}
              </p>
            </div>

            <div class="shrink-0 text-right">
              <div class="text-sm font-semibold text-gray-900 dark:text-white">
                ${{ formatCurrency(entry.value) }}
              </div>
              <div class="mt-1 text-xs text-gray-400 dark:text-dark-500">
                {{ formatShare(entry.value) }}
              </div>
            </div>
          </div>
        </div>

        <div
          v-else
          class="flex h-40 items-center justify-center px-4 text-sm text-gray-400 dark:text-dark-500"
        >
          {{ t('leaderboard.empty') }}
        </div>
      </div>
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
const medals = ['🥇', '🥈', '🥉']

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
        borderColor: chartColors.value.map(() => 'rgba(255,255,255,0.9)'),
        borderWidth: 0,
        hoverOffset: 8,
        spacing: 2,
        borderRadius: 8,
      },
    ],
  }
})

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '68%',
  layout: {
    padding: 4,
  },
  plugins: {
    legend: {
      display: false,
    },
    tooltip: {
      callbacks: {
        label: (context: any) => {
          const value = context.raw as number
          const total = totalValue.value
          const percentage = total > 0 ? ((value / total) * 100).toFixed(1) : '0.0'
          return `${context.label}: $${formatCurrency(value)} (${percentage}%)`
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
