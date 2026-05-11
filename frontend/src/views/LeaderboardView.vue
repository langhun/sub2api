<template>
  <div class="relative flex min-h-screen flex-col bg-gray-50 dark:bg-dark-950">
    <PublicPageHeader active-path="/leaderboard" :nav-link-visibility="homeNavLinkVisibility" />

    <main class="mx-auto w-full max-w-7xl flex-1 px-4 py-6 sm:px-6 sm:py-8">
      <div class="space-y-6">
        <div>
          <h1 class="text-xl font-bold text-gray-900 dark:text-white sm:text-2xl">{{ t('leaderboard.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('leaderboard.subtitle') }}</p>
        </div>

        <div
          v-if="tabs.length > 0"
          class="card p-4"
        >
          <div class="flex flex-wrap items-center gap-4">
            <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
              <button
                v-for="tab in tabs"
                :key="tab.key"
                @click="activeTab = tab.key"
                :class="[
                  'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                  activeTab === tab.key
                    ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                    : 'text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-200'
                ]"
              >
                {{ tab.label }}
              </button>
            </div>

            <div
              v-if="showPeriodSelector"
              class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-gray-700 dark:bg-dark-800"
            >
              <button
                v-for="p in periods"
                :key="p.key"
                @click="activePeriod = p.key"
                :class="[
                  'rounded-md px-2.5 py-1 text-xs font-medium transition-colors',
                  activePeriod === p.key
                    ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                    : 'text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-200'
                ]"
              >
                {{ p.label }}
              </button>
            </div>
          </div>
        </div>

        <div class="card relative overflow-hidden p-4">
          <div v-if="loading" class="absolute inset-0 z-10 flex items-center justify-center bg-white/80 backdrop-blur-sm dark:bg-dark-900/80">
            <div class="flex flex-col items-center gap-3">
              <div class="h-6 w-6 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
              <span class="text-xs text-gray-400 dark:text-dark-500">{{ t('common.loading') }}</span>
            </div>
          </div>

          <PublicConsumptionLeaderboardChart
            v-if="activeTab === 'consumption' && !loading && consumptionSummary && consumptionChartItems.length > 0"
            :chart-items="consumptionChartItems"
            :summary="consumptionSummary"
            :entries="entries"
          />

          <div v-if="!loading && tabs.length === 0" class="py-16 text-center text-sm text-gray-400 dark:text-dark-500">
            {{ t('leaderboard.tabsDisabled') }}
          </div>

          <div v-else-if="!loading && entries.length === 0" class="py-16 text-center text-sm text-gray-400 dark:text-dark-500">
            {{ t('leaderboard.empty') }}
          </div>

          <div v-else-if="activeTab !== 'consumption'">
            <div class="mb-4">
              <p class="text-sm font-semibold text-gray-900 dark:text-white">
                {{ activeTabLabel }}
              </p>
              <p v-if="showPeriodSelector" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ activePeriodLabel }}
              </p>
            </div>

            <div class="max-h-[24rem] overflow-y-auto">
              <div class="space-y-2 pr-1">
                <div
                  v-for="entry in entries"
                  :key="entry.rank"
                  class="group flex items-center gap-3 rounded-xl px-3 py-2.5 transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/50"
                >
                  <div
                    :class="rankClass(entry.rank)"
                    class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-sm font-bold"
                  >
                    <span v-if="entry.rank <= 3">{{ ['🥇', '🥈', '🥉'][entry.rank - 1] }}</span>
                    <span v-else class="text-gray-500 dark:text-dark-400">{{ entry.rank }}</span>
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ entry.username }}</p>
                    <p v-if="getSubtitle(entry)" class="mt-1 truncate text-xs text-gray-400 dark:text-dark-500">{{ getSubtitle(entry) }}</p>
                  </div>
                  <div class="shrink-0 text-right">
                    <template v-if="activeTab === 'checkin'">
                      <span class="text-sm font-bold text-amber-600 dark:text-amber-400">{{ entry.value }}</span>
                      <span class="text-xs text-amber-500/70 dark:text-amber-400/50"> {{ t('leaderboard.streakDays', { days: '' }).trim() }}</span>
                    </template>
                    <template v-else>
                      <span class="text-sm font-bold text-gray-900 dark:text-white">${{ entry.value.toFixed(2) }}</span>
                    </template>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <PublicPageFooter />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import PublicPageHeader from '@/components/common/PublicPageHeader.vue'
import PublicPageFooter from '@/components/common/PublicPageFooter.vue'
import PublicConsumptionLeaderboardChart from '@/components/leaderboard/PublicConsumptionLeaderboardChart.vue'
import {
  leaderboardAPI,
  type LeaderboardChartItem,
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

const homeNavLinkVisibility = computed(() => {
  const settings = appStore.cachedPublicSettings
  const legacyEnabled = settings?.home_nav_links_enabled !== false
  const resolve = (value?: boolean) => value ?? legacyEnabled

  return {
    leaderboard: resolve(settings?.home_nav_leaderboard_enabled),
    keyUsage: resolve(settings?.home_nav_key_usage_enabled),
    monitoring: resolve(settings?.home_nav_monitoring_enabled),
    pricing: resolve(settings?.home_nav_pricing_enabled),
  }
})

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
const activeTabLabel = computed(() => tabs.value.find((tab) => tab.key === activeTab.value)?.label ?? '')

const periods = computed(() => [
  { key: 'daily' as PeriodKey, label: t('leaderboard.periods.daily') },
  { key: 'weekly' as PeriodKey, label: t('leaderboard.periods.weekly') },
  { key: 'monthly' as PeriodKey, label: t('leaderboard.periods.monthly') },
])

const activePeriodLabel = computed(() => periods.value.find((period) => period.key === activePeriod.value)?.label ?? '')

function rankClass(rank: number): string {
  if (rank === 1) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
  if (rank === 2) return 'bg-slate-200 text-slate-700 dark:bg-slate-700/70 dark:text-slate-200'
  if (rank === 3) return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-800 dark:text-dark-300'
}

function getSubtitle(entry: LeaderboardEntry): string {
  if (activeTab.value === 'balance') {
    if (entry.extra_int) return t('leaderboard.balanceSubtitle', { count: entry.extra_int })
  } else if (activeTab.value === 'consumption') {
    if (entry.extra_int) return t('leaderboard.consumptionSubtitle', { count: entry.extra_int })
  } else if (activeTab.value === 'checkin') {
    if (entry.extra_int || entry.extra_date) {
      return t('leaderboard.checkinSubtitle', { total: entry.extra_int || 0, date: entry.extra_date || '', reward: entry.extra_float?.toFixed(2) || '0.00' })
    }
  }
  return ''
}

async function fetchData() {
  const currentFetch = ++fetchSequence
  loading.value = true
  try {
    let res
    switch (activeTab.value) {
      case 'balance':
        res = await leaderboardAPI.getBalanceLeaderboard(1, 20)
        break
      case 'consumption':
        res = await leaderboardAPI.getConsumptionLeaderboard(activePeriod.value, 1, 20)
        break
      case 'checkin':
        res = await leaderboardAPI.getCheckinLeaderboard(1, 20)
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
