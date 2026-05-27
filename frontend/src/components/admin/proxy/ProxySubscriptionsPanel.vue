<template>
  <div class="rounded-2xl border border-gray-200 bg-gray-50/70 p-3 dark:border-dark-600 dark:bg-dark-900/30">
    <div v-if="loading" class="rounded-2xl bg-white p-6 text-sm text-gray-500 shadow-sm dark:bg-dark-800 dark:text-gray-400">{{ t('common.loading') }}</div>
    <div v-else-if="items.length === 0" class="rounded-2xl bg-white p-6 text-sm text-gray-500 shadow-sm dark:bg-dark-800 dark:text-gray-400">
      {{ t('admin.proxies.subscriptions.empty') }}
    </div>
    <div v-else class="grid gap-4 xl:grid-cols-2">
      <div v-for="item in items" :key="item.id" data-test="subscription-source-card" class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <div class="flex flex-col gap-4">
          <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div class="min-w-0 space-y-2">
              <div class="flex flex-wrap items-center gap-2">
                <span class="font-medium text-gray-900 dark:text-white">{{ item.name }}</span>
                <span :class="['badge', item.enabled ? 'badge-success' : 'badge-gray']">
                  {{ item.enabled ? t('common.enabled') : t('common.disabled') }}
                </span>
                <span class="badge badge-gray">{{ item.source_format }}</span>
                <span
                  :class="[
                    'inline-flex items-center rounded-full px-2.5 py-1 text-[11px] font-medium',
                    item.last_error
                      ? 'bg-red-50 text-red-600 dark:bg-red-950/30 dark:text-red-300'
                      : item.last_success_at
                        ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/30 dark:text-emerald-300'
                        : 'bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300'
                  ]"
                >
                  {{ item.last_error
                    ? t('admin.proxies.subscriptions.statusError')
                    : item.last_success_at
                      ? t('admin.proxies.subscriptions.statusHealthy')
                      : t('admin.proxies.subscriptions.statusPending') }}
                </span>
              </div>
              <div class="rounded-xl bg-gray-50 px-3 py-2 font-mono text-xs text-gray-500 dark:bg-dark-700/70 dark:text-gray-300">
                {{ item.url }}
              </div>
            </div>
            <div class="shrink-0 rounded-xl bg-gray-50 px-3 py-2 text-xs leading-5 text-gray-500 dark:bg-dark-700/70 dark:text-gray-300">
              <div>{{ t('admin.proxies.subscriptions.targetEntryCount', { count: item.target_entry_count }) }}</div>
              <div>{{ t('admin.proxies.subscriptions.materializedCount', { count: item.last_materialized_proxy_count }) }}</div>
            </div>
          </div>

          <div class="grid gap-3 sm:grid-cols-2 2xl:grid-cols-4">
            <div class="rounded-xl border border-gray-100 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-700/60">
              <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.proxies.subscriptions.refreshInterval', { hours: item.refresh_interval_hours }) }}</div>
            </div>
            <div class="rounded-xl border border-gray-100 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-700/60">
              <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.proxies.subscriptions.nodeCount', { count: item.last_node_count }) }}</div>
            </div>
            <div class="rounded-xl border border-gray-100 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-700/60">
              <div class="text-[11px] text-gray-500 dark:text-gray-400">
                {{ item.last_refreshed_at
                  ? t('admin.proxies.subscriptions.lastRefreshed', { time: item.last_refreshed_at })
                  : t('admin.proxies.subscriptions.lastRefreshedEmpty') }}
              </div>
            </div>
            <div class="rounded-xl border border-gray-100 bg-gray-50 px-3 py-2 dark:border-dark-700 dark:bg-dark-700/60">
              <div class="text-[11px] text-gray-500 dark:text-gray-400">
                {{ item.last_success_at
                  ? t('admin.proxies.subscriptions.lastSuccess', { time: item.last_success_at })
                  : t('admin.proxies.subscriptions.lastSuccessEmpty') }}
              </div>
            </div>
          </div>

          <div v-if="item.last_error" class="rounded-xl border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-600 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300">
            {{ item.last_error }}
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <button data-test="subscription-refresh" class="btn btn-secondary" @click="$emit('refresh', item.id)">{{ t('admin.proxies.subscriptions.refreshNow') }}</button>
            <button data-test="subscription-edit" class="btn btn-secondary" @click="$emit('edit', item)">{{ t('common.edit') }}</button>
            <button data-test="subscription-view-nodes" class="btn btn-secondary" @click="$emit('view-nodes', item.id)">{{ t('admin.proxies.subscriptions.viewNodes') }}</button>
            <button data-test="subscription-delete" class="btn btn-danger" @click="$emit('delete', item.id)">{{ t('common.delete') }}</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { ProxySubscriptionSource } from '@/types'

defineProps<{
  loading: boolean
  items: ProxySubscriptionSource[]
}>()

defineEmits<{
  (e: 'refresh', id: number): void
  (e: 'edit', item: ProxySubscriptionSource): void
  (e: 'view-nodes', id: number): void
  (e: 'delete', id: number): void
}>()

const { t } = useI18n()
</script>
