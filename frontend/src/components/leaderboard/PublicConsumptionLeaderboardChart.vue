<template>
  <section class="px-4 py-4 sm:px-6">
    <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
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

    <div class="flex flex-col items-center justify-center">
      <div class="h-56 w-full max-w-[280px]">
        <Doughnut v-if="chartData" :data="chartData" :options="doughnutOptions" />
      </div>
      <p class="mt-3 text-xs text-gray-400 dark:text-dark-500">
        {{ t('leaderboard.hoverHint') }}
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArcElement, Chart as ChartJS, Legend, Tooltip } from 'chart.js'
import { Doughnut } from 'vue-chartjs'

import type { LeaderboardChartItem, LeaderboardSummary } from '@/api/leaderboard'
import { createConsumptionLeaderboardPalette } from './consumptionChartPalette'

ChartJS.register(ArcElement, Tooltip, Legend)

const props = defineProps<{
  chartItems: LeaderboardChartItem[]
  summary?: LeaderboardSummary | null
}>()

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
  cutout: '65%',
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
