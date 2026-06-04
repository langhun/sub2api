<template>
  <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <div class="stat-card">
      <div class="stat-icon stat-icon-primary">
        <Icon name="document" size="md" class="text-current" />
      </div>
      <div>
        <p class="stat-label text-xs font-medium">{{ t('usage.totalRequests') }}</p>
        <p class="stat-value">{{ stats?.total_requests?.toLocaleString() || '0' }}</p>
        <p class="stat-label text-xs">{{ t('usage.inSelectedRange') }}</p>
      </div>
    </div>
    <div class="stat-card">
      <div class="stat-icon stat-icon-warning"><svg class="h-5 w-5 text-current" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="m21 7.5-9-5.25L3 7.5m18 0-9 5.25m9-5.25v9l-9 5.25M3 7.5l9 5.25M3 7.5v9l9 5.25m0-9v9" /></svg></div>
      <div>
        <p class="stat-label text-xs font-medium">{{ t('usage.totalTokens') }}</p>
        <p class="stat-value">{{ formatTokens(stats?.total_tokens || 0) }}</p>
        <p class="stat-label text-xs">
          {{ t('usage.in') }}: {{ formatTokens(stats?.total_input_tokens || 0) }} /
          {{ t('usage.out') }}: {{ formatTokens(stats?.total_output_tokens || 0) }}
        </p>
      </div>
    </div>
    <div class="stat-card">
      <div class="stat-icon stat-icon-success">
        <Icon name="dollar" size="md" class="text-current" />
      </div>
      <div class="min-w-0 flex-1">
        <p class="stat-label text-xs font-medium">{{ t('usage.totalCost') }}</p>
        <p class="stat-value stat-trend-up">
          ${{ (stats?.total_actual_cost || 0).toFixed(4) }}
        </p>
        <p class="stat-label text-xs">
          <span class="text-amber-600 dark:text-amber-400">{{ t('usage.accountCost') }} ${{ (stats?.total_account_cost || 0).toFixed(4) }}</span>
          <span> · </span>
          <span>{{ t('usage.standardCost') }} ${{ (stats?.total_cost || 0).toFixed(4) }}</span>
        </p>
      </div>
    </div>
    <div class="stat-card">
      <div class="feature-icon feature-icon-purple h-9 w-9 rounded-xl">
        <Icon name="clock" size="md" class="text-current" />
      </div>
      <div><p class="stat-label text-xs font-medium">{{ t('usage.avgDuration') }}</p><p class="stat-value">{{ formatDuration(stats?.average_duration_ms || 0) }}</p></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { AdminUsageStatsResponse } from '@/api/admin/usage'
import Icon from '@/components/icons/Icon.vue'

defineProps<{ stats: AdminUsageStatsResponse | null }>()

const { t } = useI18n()

const formatDuration = (ms: number) =>
  ms < 1000 ? `${ms.toFixed(0)}ms` : `${(ms / 1000).toFixed(2)}s`

const formatTokens = (value: number) => {
  if (value >= 1e9) return (value / 1e9).toFixed(2) + 'B'
  if (value >= 1e6) return (value / 1e6).toFixed(2) + 'M'
  if (value >= 1e3) return (value / 1e3).toFixed(2) + 'K'
  return value.toLocaleString()
}
</script>
