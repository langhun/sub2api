<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <header><h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('leaderboard.title') }}</h1><p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('leaderboard.subtitle') }}</p></header>
      <section class="card overflow-hidden p-0">
        <div class="flex overflow-x-auto border-b border-gray-100 p-1 dark:border-dark-700" role="tablist">
          <button v-for="tab in tabs" :key="tab.key" role="tab" :aria-selected="activeTab === tab.key" class="min-w-fit flex-1 px-4 py-3 text-sm font-medium" :class="activeTab === tab.key ? 'border-b-2 border-primary-500 text-primary-600 dark:text-primary-400' : 'text-gray-500'" @click="selectTab(tab.key)">{{ tab.label }}</button>
        </div>
        <div v-if="periodic" class="flex gap-2 border-b border-gray-100 px-4 py-3 dark:border-dark-700"><button v-for="period in periods" :key="period.key" class="rounded-md px-3 py-1.5 text-xs font-medium" :class="activePeriod === period.key ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300' : 'text-gray-500 hover:bg-gray-50 dark:hover:bg-dark-800'" @click="selectPeriod(period.key)">{{ period.label }}</button></div>
        <div v-if="loading" class="flex min-h-72 items-center justify-center"><LoadingSpinner /></div>
        <div v-else-if="error" class="py-16 text-center"><Icon name="exclamationCircle" size="xl" class="mx-auto text-red-500" /><p class="mt-3 text-sm text-red-600 dark:text-red-400">{{ error }}</p><button class="btn btn-secondary mt-4" @click="fetchData">{{ t('common.retry') }}</button></div>
        <div v-else-if="entries.length === 0" class="py-16 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('leaderboard.empty') }}</div>
        <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
          <div v-for="entry in entries" :key="`${activeTab}-${entry.rank}-${entry.username}`" class="flex items-center gap-4 px-4 py-4 sm:px-6">
            <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full font-semibold" :class="rankClass(entry.rank)"><Icon v-if="entry.rank <= 3" name="badge" size="md" /><span v-else>{{ entry.rank }}</span></div>
            <div class="min-w-0 flex-1"><p class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ entry.username }}</p><p v-if="subtitle(entry)" class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">{{ subtitle(entry) }}</p></div>
            <div class="shrink-0 text-right"><p class="font-semibold text-gray-900 dark:text-white">{{ valueLabel(entry) }}</p><p class="mt-1 text-xs text-gray-500">{{ t('leaderboard.rank', { rank: entry.rank }) }}</p></div>
          </div>
        </div>
        <Pagination v-if="total > pageSize" :page="page" :page-size="pageSize" :total="total" :show-page-size-selector="false" @update:page="changePage" />
      </section>
      <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('leaderboard.privacyNotice') }}</p>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'; import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'; import LoadingSpinner from '@/components/common/LoadingSpinner.vue'; import Pagination from '@/components/common/Pagination.vue'; import Icon from '@/components/icons/Icon.vue'
import { leaderboardAPI, type LeaderboardData, type LeaderboardEntry } from '@/api/leaderboard'; import { useAppStore } from '@/stores/app'
type TabKey = 'balance' | 'consumption' | 'checkin' | 'transfer'; type PeriodKey = 'daily' | 'weekly' | 'monthly'
const { t } = useI18n(); const appStore = useAppStore(); const activeTab = ref<TabKey>('balance'); const activePeriod = ref<PeriodKey>('daily'); const entries = ref<LeaderboardEntry[]>([]); const loading = ref(false); const error = ref(''); const page = ref(1); const pageSize = 20; const total = ref(0)
const tabs = computed(() => { const settings = appStore.cachedPublicSettings; return [{ key: 'balance' as const, label: t('leaderboard.tabs.balance'), enabled: settings?.leaderboard_balance_enabled !== false }, { key: 'consumption' as const, label: t('leaderboard.tabs.consumption'), enabled: settings?.leaderboard_consumption_enabled !== false }, { key: 'checkin' as const, label: t('leaderboard.tabs.checkin'), enabled: settings?.leaderboard_checkin_enabled !== false }, { key: 'transfer' as const, label: t('leaderboard.tabs.transfer'), enabled: settings?.leaderboard_transfer_enabled ?? settings?.transfer_enabled ?? false }].filter(item => item.enabled) })
const periods = computed(() => [{ key: 'daily' as const, label: t('leaderboard.periods.daily') }, { key: 'weekly' as const, label: t('leaderboard.periods.weekly') }, { key: 'monthly' as const, label: t('leaderboard.periods.monthly') }]); const periodic = computed(() => activeTab.value === 'consumption' || activeTab.value === 'transfer')
function rankClass(rank: number) { if (rank === 1) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'; if (rank === 2) return 'bg-gray-200 text-gray-700 dark:bg-dark-700 dark:text-dark-200'; if (rank === 3) return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'; return 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-400' }
function valueLabel(entry: LeaderboardEntry) { return activeTab.value === 'checkin' ? t('leaderboard.daysValue', { value: entry.value }) : `$${entry.value.toFixed(2)}` }
function subtitle(entry: LeaderboardEntry) { if (activeTab.value === 'balance' && entry.extra_int != null) return t('leaderboard.balanceSubtitle', { count: entry.extra_int }); if (activeTab.value === 'consumption' && entry.extra_int != null) return t('leaderboard.consumptionSubtitle', { count: entry.extra_int }); if (activeTab.value === 'checkin') return t('leaderboard.checkinSubtitle', { total: entry.extra_int || 0, date: entry.extra_date || '-', reward: entry.extra_float?.toFixed(2) || '0.00' }); if (activeTab.value === 'transfer' && entry.extra_int != null) return t('leaderboard.transferSubtitle', { count: entry.extra_int }); return '' }
async function fetchData() { loading.value = true; error.value = ''; try { let result: LeaderboardData; if (activeTab.value === 'balance') result = await leaderboardAPI.getBalanceLeaderboard(page.value, pageSize); else if (activeTab.value === 'consumption') result = await leaderboardAPI.getConsumptionLeaderboard(activePeriod.value, page.value, pageSize); else if (activeTab.value === 'checkin') result = await leaderboardAPI.getCheckinLeaderboard(page.value, pageSize); else result = await leaderboardAPI.getTransferLeaderboard(activePeriod.value, page.value, pageSize); entries.value = result.items || []; total.value = result.total || 0 } catch { entries.value = []; total.value = 0; error.value = t('leaderboard.loadFailed') } finally { loading.value = false } }
function selectTab(tab: TabKey) { activeTab.value = tab; page.value = 1 } function selectPeriod(period: PeriodKey) { activePeriod.value = period; page.value = 1 } function changePage(value: number) { page.value = value; void fetchData() }
watch([activeTab, activePeriod], fetchData); onMounted(async () => { if (!appStore.publicSettingsLoaded) await appStore.fetchPublicSettings(); const first = tabs.value[0]; if (first && !tabs.value.some(tab => tab.key === activeTab.value)) activeTab.value = first.key; if (first) await fetchData() })
</script>
