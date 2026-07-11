<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl space-y-5">
      <header class="flex items-start gap-3">
        <div
          class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-600 dark:bg-primary-900/25 dark:text-primary-400"
        >
          <Icon name="chartBar" size="md" />
        </div>
        <div class="min-w-0">
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">
            {{ t('leaderboard.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">
            {{ t('leaderboard.subtitle') }}
          </p>
        </div>
      </header>

      <section
        v-if="tabs.length > 0"
        class="card flex flex-col gap-2 p-2 sm:flex-row sm:items-center sm:justify-between sm:p-3"
        :aria-label="t('leaderboard.filterLabel')"
      >
        <div class="flex min-w-0 overflow-x-auto" role="tablist">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            role="tab"
            :aria-selected="activeTab === tab.key"
            class="flex min-w-fit items-center gap-2 rounded-xl px-3.5 py-2.5 text-sm font-medium transition-colors sm:px-4"
            :class="
              activeTab === tab.key
                ? 'bg-primary-500 text-white shadow-sm shadow-primary-500/20'
                : 'text-gray-500 hover:bg-gray-50 hover:text-gray-900 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white'
            "
            @click="selectTab(tab.key)"
          >
            <Icon :name="tab.icon" size="sm" />
            {{ tab.label }}
          </button>
        </div>

        <div
          v-if="periodic"
          class="flex shrink-0 items-center gap-1 border-t border-gray-100 pt-2 dark:border-dark-700 sm:border-l sm:border-t-0 sm:pl-3 sm:pt-0"
          :aria-label="t('leaderboard.periodFilterLabel')"
        >
          <button
            v-for="period in periods"
            :key="period.key"
            type="button"
            class="flex-1 rounded-lg px-3 py-2 text-xs font-semibold transition-colors sm:flex-none"
            :class="
              activePeriod === period.key
                ? 'bg-primary-50 text-primary-700 ring-1 ring-inset ring-primary-200 dark:bg-primary-900/25 dark:text-primary-300 dark:ring-primary-800'
                : 'text-gray-500 hover:bg-gray-50 hover:text-gray-800 dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-dark-100'
            "
            @click="selectPeriod(period.key)"
          >
            {{ period.label }}
          </button>
        </div>
      </section>

      <section v-if="tabs.length === 0 && !loading" class="card py-16 text-center">
        <div
          class="mx-auto flex h-12 w-12 items-center justify-center rounded-xl bg-gray-100 text-gray-400 dark:bg-dark-800 dark:text-dark-400"
        >
          <Icon name="chartBar" size="lg" />
        </div>
        <h2 class="mt-4 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('leaderboard.noBoards') }}
        </h2>
        <p class="mx-auto mt-1 max-w-sm text-sm text-gray-500 dark:text-dark-400">
          {{ t('leaderboard.noBoardsHint') }}
        </p>
      </section>

      <section v-else class="card overflow-hidden">
        <div
          class="flex flex-col gap-4 border-b border-gray-100 px-4 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between sm:px-6"
        >
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('leaderboard.distributionTitle', { board: activeTabLabel }) }}
            </h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
              {{ t('leaderboard.distributionHint') }}
            </p>
          </div>

          <dl class="grid grid-cols-2 divide-x divide-gray-200 rounded-xl bg-gray-50 dark:divide-dark-700 dark:bg-dark-900/60">
            <div class="min-w-0 px-4 py-2.5 sm:min-w-36">
              <dt class="text-[11px] font-medium text-gray-500 dark:text-dark-400">
                {{ t('leaderboard.pageTotal') }}
              </dt>
              <dd class="mt-0.5 truncate text-base font-bold text-gray-900 dark:text-white">
                {{ compactSummaryValue }}
              </dd>
            </div>
            <div class="min-w-0 px-4 py-2.5 sm:min-w-28">
              <dt class="text-[11px] font-medium text-gray-500 dark:text-dark-400">
                {{ t('leaderboard.listedUsers') }}
              </dt>
              <dd class="mt-0.5 truncate text-base font-bold text-gray-900 dark:text-white">
                {{ formatInteger(total) }}
              </dd>
            </div>
          </dl>
        </div>

        <div v-if="loading" class="flex min-h-[30rem] items-center justify-center">
          <LoadingSpinner />
        </div>

        <div v-else-if="error" class="flex min-h-[30rem] flex-col items-center justify-center px-6 text-center">
          <div
            class="flex h-12 w-12 items-center justify-center rounded-xl bg-red-50 text-red-500 dark:bg-red-900/20 dark:text-red-400"
          >
            <Icon name="exclamationCircle" size="lg" />
          </div>
          <p class="mt-4 text-sm font-medium text-red-600 dark:text-red-400">{{ error }}</p>
          <button type="button" class="btn btn-secondary btn-sm mt-4" @click="fetchData">
            <Icon name="refresh" size="sm" />
            {{ t('common.retry') }}
          </button>
        </div>

        <div v-else-if="entries.length === 0" class="flex min-h-[30rem] flex-col items-center justify-center px-6 text-center">
          <div
            class="flex h-12 w-12 items-center justify-center rounded-xl bg-gray-100 text-gray-400 dark:bg-dark-800 dark:text-dark-400"
          >
            <Icon name="users" size="lg" />
          </div>
          <p class="mt-4 text-sm font-medium text-gray-700 dark:text-dark-200">
            {{ t('leaderboard.empty') }}
          </p>
        </div>

        <div v-else class="grid min-w-0 lg:grid-cols-[minmax(17rem,0.72fr)_minmax(0,1.8fr)]">
          <div
            class="order-2 flex min-w-0 flex-col items-center justify-start border-t border-gray-100 px-5 py-6 dark:border-dark-700 lg:order-1 lg:border-r lg:border-t-0 lg:py-7"
          >
            <div class="relative aspect-square w-full max-w-[13rem] sm:max-w-[15.5rem]">
              <Doughnut :data="chartData" :options="chartOptions" />
              <div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center px-14 text-center">
                <span class="max-w-full truncate text-xl font-bold text-gray-900 dark:text-white">
                  {{ compactSummaryValue }}
                </span>
                <span class="mt-1 text-[11px] text-gray-500 dark:text-dark-400">
                  {{ t('leaderboard.currentPageUsers', { count: entries.length }) }}
                </span>
              </div>
            </div>
            <p class="mt-5 max-w-xs text-center text-xs leading-5 text-gray-500 dark:text-dark-400">
              {{ t('leaderboard.chartHint') }}
            </p>
          </div>

          <div class="order-1 min-w-0 lg:order-2">
            <div class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700 sm:px-5">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ t('leaderboard.rankingTitle') }}
              </h3>
              <span class="text-xs text-gray-400 dark:text-dark-500">
                {{ t('leaderboard.currentPage', { page }) }}
              </span>
            </div>

            <div
              class="hidden grid-cols-[minmax(0,1fr)_7rem_8rem_4.5rem] items-center gap-3 border-b border-gray-100 bg-gray-50/70 px-5 py-2.5 text-[11px] font-medium text-gray-500 dark:border-dark-700 dark:bg-dark-900/35 dark:text-dark-400 sm:grid"
            >
              <span>{{ t('leaderboard.columns.user') }}</span>
              <span class="text-right">{{ activityHeader }}</span>
              <span class="text-right">{{ valueHeader }}</span>
              <span class="text-right">{{ t('leaderboard.columns.share') }}</span>
            </div>

            <ol class="divide-y divide-gray-100 dark:divide-dark-700">
              <li
                v-for="(entry, index) in entries"
                :key="`${activeTab}-${entry.rank}-${index}`"
                class="grid min-h-[4.5rem] grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-4 py-3 transition-colors hover:bg-gray-50/70 dark:hover:bg-dark-800/35 sm:grid-cols-[minmax(0,1fr)_7rem_8rem_4.5rem] sm:px-5"
              >
                <div class="flex min-w-0 items-center gap-3">
                  <div
                    class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl text-xs font-bold"
                    :class="rankClass(entry.rank)"
                    :aria-label="t('leaderboard.rank', { rank: entry.rank })"
                  >
                    <span>{{ entry.rank }}</span>
                  </div>
                  <span
                    class="h-2 w-2 shrink-0 rounded-full"
                    :style="{ backgroundColor: entryColor(index) }"
                    aria-hidden="true"
                  />
                  <div class="min-w-0">
                    <p class="truncate text-sm font-medium text-gray-900 dark:text-white">
                      {{ entry.username }}
                    </p>
                    <p
                      v-if="detailLabel(entry)"
                      class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400"
                      :class="activeTab === 'checkin' ? '' : 'sm:hidden'"
                    >
                      {{ detailLabel(entry) }}
                    </p>
                  </div>
                </div>

                <span class="hidden text-right text-sm tabular-nums text-gray-600 dark:text-dark-300 sm:block">
                  {{ activityValue(entry) }}
                </span>
                <span
                  class="min-w-0 truncate text-right text-sm font-semibold tabular-nums text-primary-600 dark:text-primary-400"
                  :title="preciseValueLabel(entry)"
                >
                  {{ valueLabel(entry) }}
                </span>
                <span class="hidden text-right text-xs tabular-nums text-gray-500 dark:text-dark-400 sm:block">
                  {{ percentageLabel(entry) }}
                </span>
              </li>
            </ol>
          </div>
        </div>

        <Pagination
          v-if="total > pageSize"
          :page="page"
          :page-size="pageSize"
          :total="total"
          :show-page-size-selector="false"
          @update:page="changePage"
        />
      </section>

      <p v-if="tabs.length > 0" class="flex items-start gap-2 text-xs leading-5 text-gray-500 dark:text-dark-400">
        <Icon name="infoCircle" size="sm" class="mt-0.5 shrink-0" />
        <span>{{ t('leaderboard.privacyNotice') }}</span>
      </p>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArcElement, Chart as ChartJS, Tooltip } from 'chart.js'
import type { ChartData, ChartOptions, TooltipItem } from 'chart.js'
import { Doughnut } from 'vue-chartjs'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import Pagination from '@/components/common/Pagination.vue'
import Icon from '@/components/icons/Icon.vue'
import { leaderboardAPI, type LeaderboardData, type LeaderboardEntry } from '@/api/leaderboard'
import { useAppStore } from '@/stores/app'

ChartJS.register(ArcElement, Tooltip)

type TabKey = 'balance' | 'consumption' | 'checkin' | 'transfer'
type PeriodKey = 'daily' | 'weekly' | 'monthly'

const chartColors = ['#14b8a6', '#3b82f6', '#f59e0b', '#10b981', '#f43f5e', '#8b5cf6', '#06b6d4']
const remainderColor = '#94a3b8'

const { t, locale } = useI18n()
const appStore = useAppStore()
const activeTab = ref<TabKey>('balance')
const activePeriod = ref<PeriodKey>('daily')
const entries = ref<LeaderboardEntry[]>([])
const loading = ref(false)
const error = ref('')
const page = ref(1)
const pageSize = 20
const total = ref(0)
let requestSequence = 0
let ready = false

const tabs = computed(() => {
  const settings = appStore.cachedPublicSettings
  return [
    {
      key: 'balance' as const,
      label: t('leaderboard.tabs.balance'),
      icon: 'dollar' as const,
      enabled: settings?.leaderboard_balance_enabled !== false,
    },
    {
      key: 'consumption' as const,
      label: t('leaderboard.tabs.consumption'),
      icon: 'chartBar' as const,
      enabled: settings?.leaderboard_consumption_enabled !== false,
    },
    {
      key: 'checkin' as const,
      label: t('leaderboard.tabs.checkin'),
      icon: 'calendar' as const,
      enabled: settings?.leaderboard_checkin_enabled !== false,
    },
    {
      key: 'transfer' as const,
      label: t('leaderboard.tabs.transfer'),
      icon: 'swap' as const,
      enabled: settings?.leaderboard_transfer_enabled ?? settings?.transfer_enabled ?? false,
    },
  ].filter((item) => item.enabled)
})

const periods = computed(() => [
  { key: 'daily' as const, label: t('leaderboard.periods.daily') },
  { key: 'weekly' as const, label: t('leaderboard.periods.weekly') },
  { key: 'monthly' as const, label: t('leaderboard.periods.monthly') },
])

const periodic = computed(() => activeTab.value === 'consumption' || activeTab.value === 'transfer')
const activeTabLabel = computed(() => tabs.value.find((tab) => tab.key === activeTab.value)?.label ?? '')
const summaryValue = computed(() => entries.value.reduce((sum, entry) => sum + finiteNumber(entry.value), 0))
const compactSummaryValue = computed(() => formatCompactValue(summaryValue.value))
const activityHeader = computed(() => t(`leaderboard.activityHeaders.${activeTab.value}`))
const valueHeader = computed(() => t(`leaderboard.valueHeaders.${activeTab.value}`))

const chartEntries = computed(() => {
  const visible = entries.value.slice(0, chartColors.length)
  const remainder = entries.value.slice(chartColors.length).reduce((sum, entry) => sum + finiteNumber(entry.value), 0)
  const labels = visible.map((entry) => entry.username)
  const values = visible.map((entry) => finiteNumber(entry.value))

  if (remainder > 0) {
    labels.push(t('leaderboard.others'))
    values.push(remainder)
  }

  return { labels, values }
})

const chartData = computed<ChartData<'doughnut'>>(() => ({
  labels: chartEntries.value.labels,
  datasets: [
    {
      data: chartEntries.value.values,
      backgroundColor: [...chartColors, remainderColor].slice(0, chartEntries.value.values.length),
      borderWidth: 0,
      hoverOffset: 4,
    },
  ],
}))

const chartOptions = computed<ChartOptions<'doughnut'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '68%',
  plugins: {
    legend: { display: false },
    tooltip: {
      displayColors: true,
      callbacks: {
        label: (context: TooltipItem<'doughnut'>) => {
          const value = finiteNumber(context.raw)
          const percentage = summaryValue.value > 0 ? (value / summaryValue.value) * 100 : 0
          return `${context.label}: ${formatValue(value)} (${formatPercentage(percentage)})`
        },
      },
    },
  },
}))

function finiteNumber(value: unknown) {
  const numberValue = Number(value)
  return Number.isFinite(numberValue) ? numberValue : 0
}

function numberFormatter(options: Intl.NumberFormatOptions) {
  return new Intl.NumberFormat(locale.value, options)
}

function formatInteger(value: number) {
  return numberFormatter({ maximumFractionDigits: 0 }).format(finiteNumber(value))
}

function formatCurrency(value: number, compact = false) {
  const safeValue = finiteNumber(value)
  const formatted = numberFormatter({
    notation: compact ? 'compact' : 'standard',
    minimumFractionDigits: compact ? 0 : 2,
    maximumFractionDigits: compact ? 2 : activeTab.value === 'transfer' ? 4 : 2,
  }).format(Math.abs(safeValue))
  return `${safeValue < 0 ? '-' : ''}$${formatted}`
}

function formatValue(value: number) {
  return activeTab.value === 'checkin'
    ? t('leaderboard.daysValue', { value: formatInteger(value) })
    : formatCurrency(value)
}

function formatCompactValue(value: number) {
  return activeTab.value === 'checkin'
    ? t('leaderboard.daysValue', { value: formatInteger(value) })
    : formatCurrency(value, true)
}

function formatPercentage(value: number) {
  if (value > 0 && value < 0.1) return '<0.1%'
  return `${value.toFixed(1)}%`
}

function rankClass(rank: number) {
  if (rank === 1) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (rank === 2) return 'bg-slate-100 text-slate-600 dark:bg-slate-700/60 dark:text-slate-200'
  if (rank === 3) return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
  return 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-400'
}

function entryColor(index: number) {
  return index < chartColors.length ? chartColors[index] : remainderColor
}

function valueLabel(entry: LeaderboardEntry) {
  const value = finiteNumber(entry.value)
  if (activeTab.value === 'checkin') return formatValue(value)
  return formatCurrency(value, Math.abs(value) >= 1_000_000)
}

function preciseValueLabel(entry: LeaderboardEntry) {
  return formatValue(finiteNumber(entry.value))
}

function activityValue(entry: LeaderboardEntry) {
  return formatInteger(entry.extra_int ?? 0)
}

function detailLabel(entry: LeaderboardEntry) {
  if (activeTab.value === 'balance') {
    return t('leaderboard.balanceSubtitle', { count: entry.extra_int ?? 0 })
  }
  if (activeTab.value === 'consumption') {
    return t('leaderboard.consumptionSubtitle', { count: entry.extra_int ?? 0 })
  }
  if (activeTab.value === 'checkin') {
    return t('leaderboard.checkinSubtitle', {
      total: entry.extra_int ?? 0,
      date: entry.extra_date || '-',
      reward: finiteNumber(entry.extra_float).toFixed(2),
    })
  }
  return t('leaderboard.transferSubtitle', { count: entry.extra_int ?? 0 })
}

function percentageLabel(entry: LeaderboardEntry) {
  const percentage = summaryValue.value > 0 ? (finiteNumber(entry.value) / summaryValue.value) * 100 : 0
  return formatPercentage(percentage)
}

async function fetchData() {
  if (!tabs.value.some((tab) => tab.key === activeTab.value)) return

  const requestId = ++requestSequence
  loading.value = true
  error.value = ''

  try {
    let result: LeaderboardData
    if (activeTab.value === 'balance') {
      result = await leaderboardAPI.getBalanceLeaderboard(page.value, pageSize)
    } else if (activeTab.value === 'consumption') {
      result = await leaderboardAPI.getConsumptionLeaderboard(activePeriod.value, page.value, pageSize)
    } else if (activeTab.value === 'checkin') {
      result = await leaderboardAPI.getCheckinLeaderboard(page.value, pageSize)
    } else {
      result = await leaderboardAPI.getTransferLeaderboard(activePeriod.value, page.value, pageSize)
    }

    if (requestId !== requestSequence) return
    entries.value = result.items || []
    total.value = result.total || 0
  } catch {
    if (requestId !== requestSequence) return
    entries.value = []
    total.value = 0
    error.value = t('leaderboard.loadFailed')
  } finally {
    if (requestId === requestSequence) loading.value = false
  }
}

function selectTab(tab: TabKey) {
  if (activeTab.value === tab) return
  activeTab.value = tab
  page.value = 1
}

function selectPeriod(period: PeriodKey) {
  if (activePeriod.value === period) return
  activePeriod.value = period
  page.value = 1
}

function changePage(value: number) {
  page.value = value
  void fetchData()
}

function reconcileAvailableTabs() {
  if (tabs.value.length === 0) {
    requestSequence += 1
    entries.value = []
    total.value = 0
    error.value = ''
    loading.value = false
    return false
  }

  if (!tabs.value.some((tab) => tab.key === activeTab.value)) {
    activeTab.value = tabs.value[0].key
    page.value = 1
    return false
  }

  return true
}

watch([activeTab, activePeriod], () => {
  if (ready) void fetchData()
})

watch(tabs, () => {
  if (!ready) return
  const canFetch = reconcileAvailableTabs()
  if (canFetch) void fetchData()
})

onMounted(async () => {
  if (!appStore.publicSettingsLoaded) await appStore.fetchPublicSettings()
  ready = true
  if (reconcileAvailableTabs()) await fetchData()
})
</script>
