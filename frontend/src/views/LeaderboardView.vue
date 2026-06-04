<template>
  <AppLayout>
    <main class="mx-auto w-full max-w-7xl px-0 py-0">
      <div class="space-y-6">
        <div v-if="tabs.length > 0" class="card p-4">
          <div class="flex flex-wrap items-center gap-4">
            <div class="inline-flex rounded-lg bg-[var(--muted)] p-1">
              <button
                v-for="tab in tabs"
                :key="tab.key"
                @click="activeTab = tab.key"
                :class="[
                  'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  activeTab === tab.key
                    ? 'bg-[var(--card)] text-[var(--foreground)] shadow-sm'
                    : 'text-[var(--muted-foreground)] hover:text-[var(--foreground)]'
                ]"
              >
                {{ tab.label }}
              </button>
            </div>

            <div
              v-if="showPeriodSelector"
              class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--muted)] p-0.5"
            >
              <button
                v-for="p in periods"
                :key="p.key"
                @click="activePeriod = p.key"
                :class="[
                  'rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
                  activePeriod === p.key
                    ? 'bg-[var(--card)] text-[var(--foreground)] shadow-sm'
                    : 'text-[var(--muted-foreground)] hover:text-[var(--foreground)]'
                ]"
              >
                {{ p.label }}
              </button>
            </div>
          </div>
        </div>

        <div class="card relative overflow-hidden p-4">
          <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center bg-[color-mix(in_oklch,var(--card)_90%,transparent)]">
            <div class="flex flex-col items-center gap-3">
              <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
              <span class="text-xs text-[var(--muted-foreground)]">{{ t('common.loading') }}</span>
            </div>
          </div>

          <PublicLeaderboardChart
            v-if="!loading && tabs.length > 0 && entries.length > 0"
            :chart-items="activeChartItems"
            :summary="activeSummary"
            :entries="entries"
            :title="activeChartTitle"
            :subtitle="activeChartSubtitle"
            :value-label="activeValueLabel"
            :metric-label="activeMetricLabel"
            :hover-hint="activeHoverHint"
            :value-type="activeValueType"
            :subtitle-type="activeTab"
          />

          <div v-if="!loading && tabs.length === 0" class="py-16 text-center text-sm text-gray-400 dark:text-dark-500">
            {{ t('leaderboard.tabsDisabled') }}
          </div>

          <div v-else-if="!loading && entries.length === 0" class="py-16 text-center text-sm text-gray-400 dark:text-dark-500">
            {{ t('leaderboard.empty') }}
          </div>
        </div>
      </div>
    </main>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import AppLayout from '@/components/layout/AppLayout.vue'
import PublicLeaderboardChart from '@/components/leaderboard/PublicLeaderboardChart.vue'
import {
  leaderboardAPI,
  type LeaderboardChartItem,
  type LeaderboardData,
  type LeaderboardEntry,
  type LeaderboardSummary,
} from '@/api/leaderboard'

const { t } = useI18n()
const appStore = useAppStore()

type TabKey = 'balance' | 'consumption' | 'checkin'
type PeriodKey = 'daily' | 'weekly' | 'monthly'

const activeTab = ref<TabKey>('balance')
const activePeriod = ref<PeriodKey>('daily')
const entries = ref<LeaderboardEntry[]>([])
const consumptionSummary = ref<LeaderboardSummary | null>(null)
const consumptionChartItems = ref<LeaderboardChartItem[]>([])
const loading = ref(false)
let fetchSequence = 0

const leaderboardTabVisibility = computed<Record<TabKey, boolean>>(() => {
  const settings = appStore.cachedPublicSettings
  const resolve = (value?: boolean) => value ?? true

  return {
    balance: resolve(settings?.leaderboard_balance_enabled),
    consumption: resolve(settings?.leaderboard_consumption_enabled),
    checkin: resolve(settings?.leaderboard_checkin_enabled),
  }
})

const allTabs = computed(() => [
  { key: 'balance' as TabKey, label: t('leaderboard.tabs.balance') },
  { key: 'consumption' as TabKey, label: t('leaderboard.tabs.consumption') },
  { key: 'checkin' as TabKey, label: t('leaderboard.tabs.checkin') },
])

const tabs = computed(() => allTabs.value.filter((tab) => leaderboardTabVisibility.value[tab.key] !== false))
const visibleTabKeys = computed(() => tabs.value.map((tab) => tab.key).join(','))
const showPeriodSelector = computed(() => tabs.value.length > 0 && activeTab.value === 'consumption')

const periods = computed(() => [
  { key: 'daily' as PeriodKey, label: t('leaderboard.periods.daily') },
  { key: 'weekly' as PeriodKey, label: t('leaderboard.periods.weekly') },
  { key: 'monthly' as PeriodKey, label: t('leaderboard.periods.monthly') },
])

const defaultPublicLeaderboardPageSize = 20

const activeChartItems = computed<LeaderboardChartItem[]>(() => {
  if (activeTab.value === 'consumption' && consumptionChartItems.value.length > 0) {
    return consumptionChartItems.value
  }
  return entries.value.map((entry) => ({
    username: entry.username,
    value: entry.value,
  }))
})

const activeSummary = computed<LeaderboardSummary>(() => {
  if (activeTab.value === 'consumption' && consumptionSummary.value) {
    return consumptionSummary.value
  }
  return {
    total_value: entries.value.reduce((sum, entry) => sum + entry.value, 0),
    total_users: entries.value.length,
  }
})

const activeChartTitle = computed(() => {
  if (activeTab.value === 'balance') return t('leaderboard.balanceChartTitle')
  if (activeTab.value === 'checkin') return t('leaderboard.checkinChartTitle')
  return t('leaderboard.consumptionChartTitle')
})

const activeChartSubtitle = computed(() => {
  if (activeTab.value === 'balance') return t('leaderboard.balanceChartSubtitle')
  if (activeTab.value === 'checkin') return t('leaderboard.checkinChartSubtitle')
  return t('leaderboard.consumptionChartSubtitle')
})

const activeValueLabel = computed(() => {
  if (activeTab.value === 'checkin') return t('leaderboard.streak')
  return t('leaderboard.amount')
})

const activeMetricLabel = computed(() => {
  if (activeTab.value === 'balance') return t('leaderboard.checkins')
  if (activeTab.value === 'checkin') return t('leaderboard.totalCheckins')
  return t('leaderboard.requests')
})

const activeHoverHint = computed(() => {
  if (activeTab.value === 'checkin') return t('leaderboard.checkinHoverHint')
  return t('leaderboard.hoverHint')
})

const activeValueType = computed(() => activeTab.value === 'checkin' ? 'number' : 'currency')

async function fetchData() {
  const currentFetch = ++fetchSequence
  loading.value = true
  try {
    let res: LeaderboardData
    switch (activeTab.value) {
      case 'balance':
        res = await fetchPagedLeaderboardData((pageSize) => leaderboardAPI.getBalanceLeaderboard(1, pageSize))
        break
      case 'consumption':
        res = await fetchConsumptionLeaderboardData(activePeriod.value)
        break
      case 'checkin':
        res = await fetchPagedLeaderboardData((pageSize) => leaderboardAPI.getCheckinLeaderboard(1, pageSize))
        break
    }
    if (currentFetch === fetchSequence) {
      entries.value = res.items || []
      if (activeTab.value === 'consumption') {
        consumptionSummary.value = res.summary ?? {
          total_value: 0,
          total_users: res.total || 0,
        }
        consumptionChartItems.value = res.chart_items || []
      } else {
        clearConsumptionChartData()
      }
    }
  } catch {
    if (currentFetch === fetchSequence) {
      entries.value = []
      clearConsumptionChartData()
    }
  } finally {
    if (currentFetch === fetchSequence) {
      loading.value = false
    }
  }
}

async function fetchConsumptionLeaderboardData(period: PeriodKey) {
  const firstPage = await fetchPagedLeaderboardData((pageSize) => leaderboardAPI.getConsumptionLeaderboard(period, 1, pageSize))
  const currentPageSize = firstPage.page_size || defaultPublicLeaderboardPageSize
  const totalUsers = Math.max(
    firstPage.summary?.total_users ?? 0,
    firstPage.total ?? 0,
    firstPage.chart_items?.length ?? 0,
  )

  if (totalUsers > currentPageSize) {
    return leaderboardAPI.getConsumptionLeaderboard(period, 1, totalUsers)
  }

  return firstPage
}

async function fetchPagedLeaderboardData(fetchPage: (pageSize: number) => Promise<LeaderboardData>) {
  const firstPage = await fetchPage(defaultPublicLeaderboardPageSize)
  const currentPageSize = firstPage.page_size || defaultPublicLeaderboardPageSize
  const totalItems = firstPage.total || 0

  if (totalItems > currentPageSize) {
    return fetchPage(totalItems)
  }

  return firstPage
}

function clearConsumptionChartData() {
  consumptionSummary.value = null
  consumptionChartItems.value = []
}

function clearLeaderboardData() {
  fetchSequence += 1
  entries.value = []
  clearConsumptionChartData()
  loading.value = false
}

function ensureActiveTabVisible(): boolean {
  const visibleTabs = tabs.value
  if (visibleTabs.length === 0) {
    clearLeaderboardData()
    return false
  }
  if (!visibleTabs.some((tab) => tab.key === activeTab.value)) {
    activeTab.value = visibleTabs[0].key
    return false
  }
  return true
}

function ensureActivePeriodValid(): boolean {
  if (!['daily', 'weekly', 'monthly'].includes(activePeriod.value)) {
    activePeriod.value = 'daily'
    return false
  }
  return true
}

function refreshLeaderboard() {
  if (!ensureActiveTabVisible()) return
  if (!ensureActivePeriodValid()) return
  fetchData()
}

watch([activeTab, activePeriod], () => refreshLeaderboard())
watch(visibleTabKeys, () => refreshLeaderboard())

onMounted(async () => {
  if (!appStore.publicSettingsLoaded) {
    await appStore.fetchPublicSettings()
  }
  refreshLeaderboard()
})
</script>
