<template>
  <div class="space-y-3">
    <div class="grid gap-3 lg:grid-cols-[minmax(220px,1fr)_10rem_9rem_11rem]">
      <div class="relative">
        <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500" />
        <input
          :value="searchQuery"
          type="text"
          :placeholder="t('admin.proxies.searchProxies')"
          class="input pl-10"
          @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
        />
      </div>
      <Select :model-value="filters.protocol" :options="protocolOptions" :placeholder="t('admin.proxies.allProtocols')" @update:model-value="updateFilter('protocol', $event)" @change="$emit('reload-proxies')" />
      <Select :model-value="filters.status" :options="statusOptions" :placeholder="t('admin.proxies.allStatus')" @update:model-value="updateFilter('status', $event)" @change="$emit('reload-proxies')" />
      <Select :model-value="filters.runtime_status" :options="runtimeStatusOptions" :placeholder="t('admin.proxies.allRuntimeStatus')" @update:model-value="updateFilter('runtime_status', $event)" @change="$emit('reload-proxies')" />
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <button @click="$emit('reload-proxies')" :disabled="loading" class="btn btn-secondary" :title="t('common.refresh')">
        <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
      </button>

      <button
        type="button"
        class="btn btn-secondary gap-2 border-indigo-200 bg-indigo-50 px-3 text-indigo-700 hover:bg-indigo-100 dark:border-indigo-900/60 dark:bg-indigo-900/20 dark:text-indigo-300 dark:hover:bg-indigo-900/30"
        data-test="proxy-toolbar-subscriptions"
        @click="$emit('open-subscriptions')"
      >
        <Icon name="server" size="sm" />
        <span>{{ t('admin.proxies.subscriptions.title') }}</span>
      </button>

      <button class="btn btn-secondary gap-2 px-3" data-test="proxy-toolbar-pool" @click="$emit('open-pool')">
        <Icon name="shield" size="sm" />
        <span>{{ t('admin.proxies.poolMembersAction') }}</span>
      </button>

      <div class="relative" @click.stop>
        <button
          @click.stop="$emit('toggle-tools-dropdown')"
          class="btn btn-secondary gap-2 px-3"
          :title="t('admin.proxies.toolsMenu')"
          data-test="proxy-toolbar-tools"
        >
          <Icon name="more" size="sm" />
          <span>{{ t('admin.proxies.toolsMenu') }}</span>
          <Icon name="chevronDown" size="xs" />
        </button>

        <div
          v-if="showProxyToolsDropdown"
          class="absolute right-0 z-50 mt-2 w-72 origin-top-right overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-800"
        >
          <div class="space-y-3 p-3">
            <div class="rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-700/70">
              <div class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.proxies.toolsMenu') }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">{{ toolSummaryLabel }}</div>
            </div>

            <div class="grid gap-2">
              <button
                class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-sky-300 hover:bg-sky-50/60 dark:border-gray-700 dark:hover:border-sky-700 dark:hover:bg-sky-900/10"
                data-test="proxy-toolbar-import"
                @click="$emit('open-import')"
              >
                <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-sky-50 text-sky-600 dark:bg-sky-900/30 dark:text-sky-300"><Icon name="upload" size="sm" /></span>
                <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.proxies.dataImport') }}</span>
              </button>

              <button
                class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-indigo-300 hover:bg-indigo-50/60 dark:border-gray-700 dark:hover:border-indigo-700 dark:hover:bg-indigo-900/10"
                data-test="proxy-toolbar-export"
                @click="$emit('open-export')"
              >
                <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300"><Icon name="download" size="sm" /></span>
                <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ exportButtonLabel }}</span>
              </button>

              <button
                @click.stop="$emit('toggle-column-dropdown')"
                class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-violet-300 hover:bg-violet-50/60 dark:border-gray-700 dark:hover:border-violet-700 dark:hover:bg-violet-900/10"
                :title="t('admin.users.columnSettings')"
                data-test="proxy-toolbar-columns"
              >
                <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-violet-50 text-violet-600 dark:bg-violet-900/30 dark:text-violet-300">
                  <svg class="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" />
                  </svg>
                </span>
                <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.users.columnSettings') }}</span>
              </button>
            </div>

            <div v-if="showColumnDropdown" class="rounded-xl border border-gray-200 bg-white p-2 dark:border-gray-700 dark:bg-gray-800">
              <div class="max-h-60 overflow-y-auto">
                <button
                  v-for="col in toggleableColumns"
                  :key="col.key"
                  @click.stop="$emit('toggle-column', col.key)"
                  class="flex w-full items-center justify-between rounded-md px-3 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-200 dark:hover:bg-gray-700"
                >
                  <span>{{ col.label }}</span>
                  <Icon v-if="isColumnVisible(col.key)" name="check" size="sm" class="text-primary-500" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div
        v-if="selectedCount > 0"
        class="inline-flex items-center rounded-full border border-primary-200 bg-primary-50 px-3 py-1.5 text-xs text-primary-700 dark:border-primary-900/40 dark:bg-primary-900/20 dark:text-primary-200"
      >
        {{ t('common.selectedCount', { count: selectedCount }) }} · {{ t('admin.proxies.batchBarHint') }}
      </div>

      <button @click="$emit('create-proxy')" class="btn btn-primary">
        <Icon name="plus" size="md" class="mr-2" />
        {{ t('admin.proxies.createProxy') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import type { Column } from '@/components/common/types'

const { t } = useI18n()

const props = defineProps<{
  searchQuery: string
  filters: {
    protocol: string
    status: string
    runtime_status: string
  }
  protocolOptions: Array<{ value: string; label: string }>
  statusOptions: Array<{ value: string; label: string }>
  runtimeStatusOptions: Array<{ value: string; label: string }>
  loading: boolean
  batchTesting: boolean
  batchQualityChecking: boolean
  selectedCount: number
  showColumnDropdown: boolean
  showProxyToolsDropdown: boolean
  toggleableColumns: Column[]
  isColumnVisible: (key: string) => boolean
}>()

const emit = defineEmits<{
  (e: 'update:searchQuery', value: string): void
  (e: 'update:filters', filters: { protocol: string; status: string; runtime_status: string }): void
  (e: 'reload-proxies'): void
  (e: 'toggle-column-dropdown'): void
  (e: 'toggle-tools-dropdown'): void
  (e: 'toggle-column', key: string): void
  (e: 'open-import'): void
  (e: 'open-export'): void
  (e: 'open-pool'): void
  (e: 'open-subscriptions'): void
  (e: 'create-proxy'): void
}>()

const updateFilter = (key: 'protocol' | 'status' | 'runtime_status', value: string | number | boolean | null) => {
  emit('update:filters', {
    protocol: key === 'protocol' ? String(value ?? '') : props.filters.protocol,
    status: key === 'status' ? String(value ?? '') : props.filters.status,
    runtime_status: key === 'runtime_status' ? String(value ?? '') : props.filters.runtime_status
  })
}

const exportButtonLabel = computed(() =>
  props.selectedCount > 0 ? t('admin.proxies.dataExportSelected') : t('admin.proxies.dataExport')
)

const toolSummaryLabel = computed(() =>
  props.selectedCount > 0
    ? t('admin.proxies.toolsSummarySelected', { count: props.selectedCount })
    : t('admin.proxies.toolsSummary')
)
</script>
