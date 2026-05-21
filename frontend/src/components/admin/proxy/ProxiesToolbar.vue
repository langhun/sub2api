<template>
  <div class="space-y-3">
    <div class="flex flex-wrap items-center gap-3">
      <div class="relative w-full sm:w-64">
        <Icon
          name="search"
          size="md"
          class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
        />
        <input
          :value="searchQuery"
          type="text"
          :placeholder="t('admin.proxies.searchProxies')"
          class="input pl-10"
          @input="$emit('update:searchQuery', ($event.target as HTMLInputElement).value)"
        />
      </div>

      <div class="w-full sm:w-40">
        <Select
          :model-value="filters.protocol"
          :options="protocolOptions"
          :placeholder="t('admin.proxies.allProtocols')"
          @update:model-value="updateFilter('protocol', $event)"
          @change="$emit('reload-proxies')"
        />
      </div>
      <div class="w-full sm:w-36">
        <Select
          :model-value="filters.status"
          :options="statusOptions"
          :placeholder="t('admin.proxies.allStatus')"
          @update:model-value="updateFilter('status', $event)"
          @change="$emit('reload-proxies')"
        />
      </div>
      <div class="w-full sm:w-44">
        <Select
          :model-value="filters.runtime_status"
          :options="runtimeStatusOptions"
          :placeholder="t('admin.proxies.allRuntimeStatus')"
          @update:model-value="updateFilter('runtime_status', $event)"
          @change="$emit('reload-proxies')"
        />
      </div>

      <div class="flex flex-1 flex-wrap items-center justify-end gap-2">
        <button
          @click="$emit('reload-proxies')"
          :disabled="loading"
          class="btn btn-secondary"
          :title="t('common.refresh')"
        >
          <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
        </button>

        <template>
          <div class="flex flex-wrap items-center gap-2 rounded-2xl border border-gray-200/80 bg-white/90 p-2 shadow-sm shadow-gray-100/60 dark:border-gray-700/80 dark:bg-gray-900/70 dark:shadow-black/10">
            <button
              type="button"
              class="btn btn-secondary gap-2 border-indigo-200 bg-indigo-50 px-3 text-indigo-700 hover:bg-indigo-100 dark:border-indigo-900/60 dark:bg-indigo-900/20 dark:text-indigo-300 dark:hover:bg-indigo-900/30"
              data-test="proxy-toolbar-mihomo"
              @click="$emit('open-mihomo')"
            >
              <Icon name="server" size="sm" />
              <span>{{ t('admin.proxies.projectMihomo.manage') }}</span>
            </button>

            <div class="relative" @click.stop>
              <button
                @click.stop="$emit('toggle-column-dropdown')"
                class="btn btn-secondary px-2 md:px-3"
                :title="t('admin.users.columnSettings')"
                data-test="proxy-toolbar-columns"
              >
                <svg class="h-4 w-4 md:mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z"
                  />
                </svg>
                <span class="hidden md:inline">{{ t('admin.users.columnSettings') }}</span>
              </button>
              <div
                v-if="showColumnDropdown"
                class="absolute right-0 z-50 mt-2 w-52 origin-top-right rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
              >
                <div class="max-h-80 overflow-y-auto p-2">
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

            <button
              class="btn btn-secondary gap-2 px-3"
              data-test="proxy-toolbar-pool"
              @click="$emit('open-pool')"
            >
              <Icon name="shield" size="sm" />
              <span>{{ t('admin.proxies.poolMembersAction') }}</span>
            </button>
          </div>

          <div class="relative" @click.stop>
            <button
              @click.stop="$emit('toggle-batch-dropdown')"
              class="btn btn-secondary gap-2 px-3"
              :title="t('admin.proxies.batchActionsMenu')"
              data-test="proxy-toolbar-batch-toggle"
            >
              <Icon name="more" size="sm" />
              <span>{{ t('admin.proxies.batchActionsMenu') }}</span>
              <span
                v-if="selectedCount > 0"
                class="inline-flex min-w-5 items-center justify-center rounded-full bg-primary-100 px-1.5 py-0.5 text-xs font-semibold text-primary-700 dark:bg-primary-900/40 dark:text-primary-200"
              >
                {{ selectedCount }}
              </span>
              <Icon name="chevronDown" size="xs" />
            </button>
            <div
              v-if="showProxyBatchDropdown"
              class="absolute right-0 z-50 mt-2 w-[min(24rem,calc(100vw-2rem))] origin-top-right overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-xl dark:border-gray-700 dark:bg-gray-800"
            >
              <div class="space-y-4 p-3">
                <div class="flex items-center justify-between gap-3 rounded-xl bg-gray-50 px-3 py-2 dark:bg-dark-700/70">
                  <div class="min-w-0">
                    <div class="text-sm font-semibold text-gray-900 dark:text-white">
                      {{ t('admin.proxies.batchActionsMenu') }}
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      {{ batchScopeLabel }}
                    </div>
                  </div>
                  <span
                    class="inline-flex shrink-0 items-center rounded-full px-2.5 py-1 text-xs font-medium"
                    :class="selectedCount > 0
                      ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/40 dark:text-primary-200'
                      : 'bg-gray-200 text-gray-700 dark:bg-dark-600 dark:text-gray-300'"
                  >
                    {{ selectedCount > 0 ? selectedCount : t('common.all') }}
                  </span>
                </div>

                <div class="grid gap-2 sm:grid-cols-2">
                  <button
                    class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-sky-300 hover:bg-sky-50/60 dark:border-gray-700 dark:hover:border-sky-700 dark:hover:bg-sky-900/10"
                    data-test="proxy-toolbar-import"
                    @click="$emit('open-import')"
                  >
                    <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-sky-50 text-sky-600 dark:bg-sky-900/30 dark:text-sky-300">
                      <Icon name="upload" size="sm" />
                    </span>
                    <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.proxies.dataImport') }}</span>
                  </button>

                  <button
                    class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-indigo-300 hover:bg-indigo-50/60 dark:border-gray-700 dark:hover:border-indigo-700 dark:hover:bg-indigo-900/10"
                    data-test="proxy-toolbar-export"
                    @click="$emit('open-export')"
                  >
                    <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600 dark:bg-indigo-900/30 dark:text-indigo-300">
                      <Icon name="download" size="sm" />
                    </span>
                    <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ exportButtonLabel }}</span>
                  </button>
                </div>

                <div class="grid gap-2 sm:grid-cols-2">
                  <button
                    class="rounded-xl border border-emerald-200 bg-emerald-50/70 p-3 text-left transition hover:border-emerald-300 hover:bg-emerald-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-emerald-900/50 dark:bg-emerald-900/10 dark:hover:bg-emerald-900/20"
                    data-test="proxy-toolbar-batch-test"
                    :disabled="batchTesting || loading"
                    @click="$emit('batch-test')"
                  >
                    <div class="mb-2 inline-flex h-8 w-8 items-center justify-center rounded-lg bg-white text-emerald-600 shadow-sm dark:bg-dark-700 dark:text-emerald-300">
                      <Icon name="play" size="sm" />
                    </div>
                    <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.proxies.testConnection') }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ batchScopeLabel }}</div>
                  </button>

                  <button
                    class="rounded-xl border border-blue-200 bg-blue-50/70 p-3 text-left transition hover:border-blue-300 hover:bg-blue-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-blue-900/50 dark:bg-blue-900/10 dark:hover:bg-blue-900/20"
                    data-test="proxy-toolbar-batch-quality"
                    :disabled="batchQualityChecking || loading"
                    @click="$emit('batch-quality-check')"
                  >
                    <div class="mb-2 inline-flex h-8 w-8 items-center justify-center rounded-lg bg-white text-blue-600 shadow-sm dark:bg-dark-700 dark:text-blue-300">
                      <Icon name="shield" size="sm" />
                    </div>
                    <div class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.proxies.batchQualityCheck') }}</div>
                    <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ batchScopeLabel }}</div>
                  </button>
                </div>

                <div class="grid gap-2 sm:grid-cols-2">
                  <button
                    class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-violet-300 hover:bg-violet-50/60 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:hover:border-violet-700 dark:hover:bg-violet-900/10"
                    data-test="proxy-toolbar-batch-enable-pool"
                    :disabled="selectedCount === 0"
                    @click="$emit('batch-enable-pool')"
                  >
                    <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-violet-50 text-violet-600 dark:bg-violet-900/30 dark:text-violet-300">
                      <Icon name="plus" size="sm" />
                    </span>
                    <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.proxies.poolEnableAction') }}</span>
                  </button>

                  <button
                    class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-gray-300 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:hover:border-gray-600 dark:hover:bg-dark-700/70"
                    data-test="proxy-toolbar-batch-disable-pool"
                    :disabled="selectedCount === 0"
                    @click="$emit('batch-disable-pool')"
                  >
                    <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                      <Icon name="x" size="sm" />
                    </span>
                    <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.proxies.poolDisableAction') }}</span>
                  </button>

                  <button
                    class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-amber-300 hover:bg-amber-50/60 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:hover:border-amber-700 dark:hover:bg-amber-900/10"
                    data-test="proxy-toolbar-batch-clear-cooldown"
                    :disabled="selectedCount === 0"
                    @click="$emit('batch-clear-cooldown')"
                  >
                    <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-amber-50 text-amber-600 dark:bg-amber-900/30 dark:text-amber-300">
                      <Icon name="refresh" size="sm" />
                    </span>
                    <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.proxies.clearCooldownAction') }}</span>
                  </button>

                  <button
                    class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-cyan-300 hover:bg-cyan-50/60 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:hover:border-cyan-700 dark:hover:bg-cyan-900/10"
                    data-test="proxy-toolbar-batch-assign"
                    :disabled="selectedCount === 0"
                    @click="$emit('batch-assign')"
                  >
                    <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-cyan-50 text-cyan-600 dark:bg-cyan-900/30 dark:text-cyan-300">
                      <Icon name="users" size="sm" />
                    </span>
                    <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.proxies.assignAccounts.open') }}</span>
                  </button>

                  <button
                    class="flex items-center gap-3 rounded-xl border border-gray-200 px-3 py-2.5 text-left transition hover:border-orange-300 hover:bg-orange-50/60 disabled:cursor-not-allowed disabled:opacity-50 dark:border-gray-700 dark:hover:border-orange-700 dark:hover:bg-orange-900/10"
                    data-test="proxy-toolbar-batch-unassign"
                    :disabled="selectedCount === 0"
                    @click="$emit('batch-unassign')"
                  >
                    <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-orange-50 text-orange-600 dark:bg-orange-900/30 dark:text-orange-300">
                      <Icon name="x" size="sm" />
                    </span>
                    <span class="min-w-0 flex-1 text-sm text-gray-700 dark:text-gray-200">{{ t('admin.proxies.quickUnassign') }}</span>
                  </button>

                  <button
                    class="flex items-center gap-3 rounded-xl border border-red-200 px-3 py-2.5 text-left text-red-600 transition hover:border-red-300 hover:bg-red-50/60 disabled:cursor-not-allowed disabled:opacity-50 dark:border-red-900/50 dark:text-red-400 dark:hover:border-red-800 dark:hover:bg-red-900/10"
                    data-test="proxy-toolbar-batch-delete"
                    :disabled="selectedCount === 0"
                    @click="$emit('batch-delete')"
                  >
                    <span class="inline-flex h-8 w-8 items-center justify-center rounded-lg bg-red-50 text-red-600 dark:bg-red-900/30 dark:text-red-300">
                      <Icon name="trash" size="sm" />
                    </span>
                    <span class="min-w-0 flex-1 text-sm">{{ t('admin.proxies.batchDeleteAction') }}</span>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <button @click="$emit('create-proxy')" class="btn btn-primary">
            <Icon name="plus" size="md" class="mr-2" />
            {{ t('admin.proxies.createProxy') }}
          </button>
        </template>
      </div>
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
  showProxyBatchDropdown: boolean
  toggleableColumns: Column[]
  isColumnVisible: (key: string) => boolean
}>()

const emit = defineEmits<{
  (e: 'update:searchQuery', value: string): void
  (e: 'update:filters', filters: { protocol: string; status: string; runtime_status: string }): void
  (e: 'reload-proxies'): void
  (e: 'toggle-column-dropdown'): void
  (e: 'toggle-tools-dropdown'): void
  (e: 'toggle-batch-dropdown'): void
  (e: 'toggle-column', key: string): void
  (e: 'open-import'): void
  (e: 'open-export'): void
  (e: 'open-pool'): void
  (e: 'open-mihomo'): void
  (e: 'batch-test'): void
  (e: 'batch-quality-check'): void
  (e: 'batch-enable-pool'): void
  (e: 'batch-disable-pool'): void
  (e: 'batch-clear-cooldown'): void
  (e: 'batch-assign'): void
  (e: 'batch-unassign'): void
  (e: 'batch-delete'): void
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

const batchScopeLabel = computed(() =>
  props.selectedCount > 0
    ? t('common.selectedCount', { count: props.selectedCount })
    : t('admin.proxies.batchTest')
)
</script>
