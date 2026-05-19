<template>
  <div class="rounded-xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
    <div v-if="loading" class="p-6 text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</div>
    <div v-else-if="items.length === 0" class="p-6 text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.proxies.subscriptions.empty') }}
    </div>
    <div v-else class="divide-y divide-gray-200 dark:divide-dark-600">
      <div v-for="item in items" :key="item.id" class="flex flex-col gap-3 p-4 md:flex-row md:items-center md:justify-between">
        <div class="min-w-0 space-y-1">
          <div class="flex items-center gap-2">
            <span class="font-medium text-gray-900 dark:text-white">{{ item.name }}</span>
            <span :class="['badge', item.enabled ? 'badge-success' : 'badge-gray']">
              {{ item.enabled ? t('common.enabled') : t('common.disabled') }}
            </span>
            <span class="badge badge-gray">{{ item.source_format }}</span>
          </div>
          <div class="truncate text-xs text-gray-500 dark:text-gray-400">{{ item.url }}</div>
          <div class="flex flex-wrap gap-3 text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('admin.proxies.subscriptions.refreshInterval', { hours: item.refresh_interval_hours }) }}</span>
            <span>{{ t('admin.proxies.subscriptions.targetEntryCount', { count: item.target_entry_count }) }}</span>
            <span>{{ t('admin.proxies.subscriptions.nodeCount', { count: item.last_node_count }) }}</span>
            <span>{{ t('admin.proxies.subscriptions.materializedCount', { count: item.last_materialized_proxy_count }) }}</span>
            <span v-if="item.last_refreshed_at">{{ t('admin.proxies.subscriptions.lastRefreshed', { time: item.last_refreshed_at }) }}</span>
          </div>
          <div v-if="item.last_error" class="text-xs text-red-500 dark:text-red-400">{{ item.last_error }}</div>
        </div>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" @click="$emit('refresh', item.id)">{{ t('admin.proxies.subscriptions.refreshNow') }}</button>
          <button class="btn btn-secondary" @click="$emit('edit', item)">{{ t('common.edit') }}</button>
          <button class="btn btn-secondary" @click="$emit('view-nodes', item.id)">{{ t('admin.proxies.subscriptions.viewNodes') }}</button>
          <button class="btn btn-danger" @click="$emit('delete', item.id)">{{ t('common.delete') }}</button>
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
