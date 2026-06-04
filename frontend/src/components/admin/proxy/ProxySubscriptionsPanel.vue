<template>
  <div class="overflow-hidden rounded-xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
    <div v-if="loading" class="p-6 text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</div>
    <div v-else-if="items.length === 0" class="p-6 text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.proxies.subscriptions.empty') }}
    </div>
    <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
      <div
        v-for="item in items"
        :key="item.id"
        data-test="subscription-source-card"
        class="px-4 py-3 transition hover:bg-gray-50/80 dark:hover:bg-dark-700/35"
      >
        <div class="grid gap-3 xl:grid-cols-[minmax(0,1fr)_auto] xl:items-center">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <span
                :class="[
                  'h-2.5 w-2.5 rounded-full',
                  item.last_error
                    ? 'bg-red-500'
                    : item.last_success_at
                      ? 'bg-[var(--success)]'
                      : 'bg-amber-400'
                ]"
              />
              <span class="truncate text-sm font-semibold text-gray-900 dark:text-white">{{ item.name }}</span>
              <span :class="['badge', item.enabled ? 'badge-success' : 'badge-secondary']">
                {{ item.enabled ? t('common.enabled') : t('common.disabled') }}
              </span>
              <span class="badge badge-gray">{{ item.source_format }}</span>
              <span
                :class="[
                  'inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium',
                  item.last_error
                    ? 'bg-red-50 text-red-600 dark:bg-red-950/30 dark:text-red-300'
                    : item.last_success_at
                      ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300'
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

            <div class="mt-2 truncate font-mono text-xs text-gray-500 dark:text-gray-400">
              {{ item.url }}
            </div>

            <div class="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-xs text-gray-500 dark:text-gray-400">
              <span>{{ t('admin.proxies.subscriptions.nodeCount', { count: item.last_node_count }) }}</span>
              <span>{{ t('admin.proxies.subscriptions.targetEntryCount', { count: item.target_entry_count }) }}</span>
              <span>{{ t('admin.proxies.subscriptions.materializedCount', { count: item.last_materialized_proxy_count }) }}</span>
              <span>{{ t('admin.proxies.subscriptions.refreshInterval', { hours: item.refresh_interval_hours }) }}</span>
              <span>{{ item.last_success_at ? t('admin.proxies.subscriptions.lastSuccess', { time: item.last_success_at }) : t('admin.proxies.subscriptions.lastSuccessEmpty') }}</span>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2 xl:justify-end">
            <button data-test="subscription-refresh" class="btn btn-secondary btn-sm" @click="$emit('refresh', item.id)">
              <Icon name="refresh" size="sm" class="mr-1" />
              {{ t('admin.proxies.subscriptions.refreshNow') }}
            </button>
            <button data-test="subscription-view-nodes" class="btn btn-secondary btn-sm" @click="$emit('view-nodes', item.id)">
              <Icon name="grid" size="sm" class="mr-1" />
              {{ t('admin.proxies.subscriptions.viewNodes') }}
            </button>
            <button data-test="subscription-edit" class="btn btn-secondary btn-sm" @click="$emit('edit', item)">
              <Icon name="edit" size="sm" class="mr-1" />
              {{ t('common.edit') }}
            </button>
            <button data-test="subscription-delete" class="btn btn-danger btn-sm" @click="$emit('delete', item.id)">
              <Icon name="trash" size="sm" class="mr-1" />
              {{ t('common.delete') }}
            </button>
          </div>
        </div>

        <div v-if="item.last_error" class="mt-3 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-600 dark:border-red-900/40 dark:bg-red-950/20 dark:text-red-300">
          {{ item.last_error }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
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
