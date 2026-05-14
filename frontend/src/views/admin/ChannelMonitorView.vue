<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-3">
          <MonitorFiltersBar
            v-model:search="searchQuery"
            v-model:provider="providerFilter"
            v-model:enabled="enabledFilter"
            :loading="loading"
            @reload="reload"
            @create="openCreateDialog"
            @manage-templates="showTemplateManager = true"
            @search-input="handleSearch"
          />

          <div class="flex flex-col gap-3 rounded-xl border border-gray-200 bg-white px-4 py-3 dark:border-dark-700 dark:bg-dark-900 md:flex-row md:items-center md:justify-between">
            <div class="space-y-1">
              <div class="flex flex-wrap items-center gap-2 text-sm text-gray-700 dark:text-gray-200">
                <span class="font-medium">{{ t('admin.channelMonitor.pageStatusTitle') }}</span>
                <span
                  class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium"
                  :class="monitoringEnabled
                    ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
                    : 'bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300'"
                >
                  {{ monitoringEnabled ? t('admin.channelMonitor.pageStatusEnabled') : t('admin.channelMonitor.pageStatusDisabled') }}
                </span>
                <span
                  v-if="autoRefresh.enabled.value"
                  class="text-xs tabular-nums text-gray-500 dark:text-gray-400"
                >
                  {{ t('admin.channelMonitor.autoRefreshHint', { seconds: autoRefresh.countdown.value }) }}
                </span>
              </div>
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ monitoringEnabled ? t('admin.channelMonitor.pageStatusDescription') : t('admin.channelMonitor.pageStatusDisabledDescription') }}
              </p>
            </div>

            <div class="flex items-center justify-end gap-2">
              <AutoRefreshButton
                :enabled="autoRefresh.enabled.value"
                :interval-seconds="autoRefresh.intervalSeconds.value"
                :countdown="autoRefresh.countdown.value"
                :intervals="autoRefresh.intervals"
                @update:enabled="autoRefresh.setEnabled"
                @update:interval="autoRefresh.setInterval"
              />
            </div>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="monitors" :loading="loading">
          <template #cell-name="{ row, value }">
            <div class="flex items-center gap-1.5">
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
              <HelpTooltip v-if="row.api_key_decrypt_failed" :content="t('admin.channelMonitor.apiKeyDecryptFailed')">
                <Icon name="exclamationTriangle" size="sm" class="text-red-500" />
              </HelpTooltip>
            </div>
          </template>

          <template #cell-provider="{ row }">
            <span class="inline-flex items-center rounded-md px-2 py-0.5 text-xs font-medium" :class="providerBadgeClass(row.provider)">
              {{ providerLabel(row.provider) }}
            </span>
          </template>

          <template #cell-primary_model="{ row }">
            <MonitorPrimaryModelCell :row="row" />
          </template>

          <template #cell-availability_7d="{ row }">
            <span class="text-sm text-gray-900 dark:text-gray-100">{{ formatAvailability(row) }}</span>
          </template>

          <template #cell-latest_check="{ row }">
            <div class="space-y-1">
              <div class="text-sm text-gray-900 dark:text-gray-100">{{ formatLastCheckedAt(row.last_checked_at) }}</div>
              <div class="text-xs text-gray-500 dark:text-gray-400">
                {{ row.last_checked_at ? formatRelativeTime(row.last_checked_at) : t('common.time.never') }}
              </div>
            </div>
          </template>

          <template #cell-latency="{ row }">
            <span class="text-sm text-gray-900 dark:text-gray-100">{{ formatLatency(row.primary_latency_ms) }}</span>
          </template>

          <template #cell-enabled="{ row }">
            <Toggle :modelValue="row.enabled" @update:modelValue="toggleEnabled(row)" />
          </template>

          <template #cell-actions="{ row }">
            <MonitorActionsCell
              :row="row"
              :running="runningId === row.id"
              @run="handleRunNow"
              @history="openHistoryDialog"
              @edit="openEditDialog"
              @delete="handleDelete"
            />
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.channelMonitor.noMonitorsYet')"
              :description="t('admin.channelMonitor.createFirstMonitor')"
              :action-text="t('admin.channelMonitor.createButton')"
              @action="openCreateDialog"
            />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="onPageChange"
          @update:pageSize="onPageSizeChange"
        />
      </template>
    </TablePageLayout>

    <MonitorFormDialog
      :show="showDialog"
      :monitor="editing"
      @close="closeDialog"
      @saved="reload"
    />

    <MonitorTemplateManagerDialog
      :show="showTemplateManager"
      @close="showTemplateManager = false"
      @updated="reload"
    />

    <MonitorRunResultDialog
      :show="showRunResult"
      :results="runResults"
      @close="showRunResult = false"
    />

    <BaseDialog
      :show="showHistoryDialog"
      :title="historyDialogTitle"
      width="wide"
      @close="closeHistoryDialog"
    >
      <div class="space-y-4">
        <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div>
            <p class="text-sm font-medium text-gray-900 dark:text-white">{{ t('admin.channelMonitor.historySubtitle') }}</p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.channelMonitor.historyDescription') }}</p>
          </div>

          <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
            <Select
              v-model="historyModelFilter"
              :options="historyModelOptions"
              class="w-full sm:w-56"
            />
            <button
              type="button"
              class="btn btn-secondary"
              :disabled="historyLoading"
              @click="void loadHistory()"
            >
              <Icon name="refresh" size="sm" :class="historyLoading ? 'animate-spin' : ''" />
              <span class="ml-2">{{ t('common.refresh') }}</span>
            </button>
          </div>
        </div>

        <div v-if="historyLoading" class="space-y-2">
          <div
            v-for="n in 6"
            :key="n"
            class="h-16 animate-pulse rounded-xl border border-gray-200 bg-gray-50 dark:border-dark-700 dark:bg-dark-800"
          ></div>
        </div>

        <div
          v-else-if="historyEntries.length === 0"
          class="rounded-xl border border-dashed border-gray-300 bg-gray-50 px-4 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-400"
        >
          {{ t('admin.channelMonitor.historyEmpty') }}
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="entry in historyEntries"
            :key="entry.id"
            class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
          >
            <div class="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
              <div class="min-w-0 space-y-2">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ entry.model }}</span>
                  <span
                    class="inline-flex items-center rounded-full px-2 py-0.5 text-[11px] font-medium"
                    :class="statusBadgeClass(entry.status)"
                  >
                    {{ statusLabel(entry.status) }}
                  </span>
                </div>
                <p class="break-words text-sm text-gray-600 dark:text-gray-300">{{ entry.message || '-' }}</p>
              </div>

              <div class="grid min-w-0 gap-2 text-xs text-gray-500 dark:text-gray-400 md:min-w-[240px]">
                <div class="flex items-center justify-between gap-3">
                  <span>{{ t('admin.channelMonitor.historyCheckedAt') }}</span>
                  <span class="text-right text-gray-700 dark:text-gray-200">{{ formatDateTime(entry.checked_at) }}</span>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <span>{{ t('admin.channelMonitor.historyLatency') }}</span>
                  <span class="text-right text-gray-700 dark:text-gray-200">{{ formatLatency(entry.latency_ms) }} ms</span>
                </div>
                <div class="flex items-center justify-between gap-3">
                  <span>{{ t('admin.channelMonitor.historyPingLatency') }}</span>
                  <span class="text-right text-gray-700 dark:text-gray-200">{{ formatLatency(entry.ping_latency_ms) }} ms</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end">
          <button class="btn btn-primary" @click="closeHistoryDialog">{{ t('common.close') }}</button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('common.delete')"
      :message="deleteConfirmMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { adminAPI } from '@/api/admin'
import type {
  ChannelMonitor,
  CheckResult,
  HistoryItem,
  ListParams,
  Provider,
} from '@/api/admin/channelMonitor'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import HelpTooltip from '@/components/common/HelpTooltip.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import Toggle from '@/components/common/Toggle.vue'
import MonitorFiltersBar from '@/components/admin/monitor/MonitorFiltersBar.vue'
import MonitorFormDialog from '@/components/admin/monitor/MonitorFormDialog.vue'
import MonitorTemplateManagerDialog from '@/components/admin/monitor/MonitorTemplateManagerDialog.vue'
import MonitorRunResultDialog from '@/components/admin/monitor/MonitorRunResultDialog.vue'
import MonitorPrimaryModelCell from '@/components/admin/monitor/MonitorPrimaryModelCell.vue'
import MonitorActionsCell from '@/components/admin/monitor/MonitorActionsCell.vue'
import AutoRefreshButton from '@/components/common/AutoRefreshButton.vue'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { useAutoRefresh } from '@/composables/useAutoRefresh'
import { useChannelMonitorFormat } from '@/composables/useChannelMonitorFormat'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const {
  providerLabel,
  providerBadgeClass,
  formatLatency,
  formatAvailability,
  formatRelativeTime,
  statusLabel,
  statusBadgeClass,
} = useChannelMonitorFormat()

const monitors = ref<ChannelMonitor[]>([])
const loading = ref(false)
const runningId = ref<number | null>(null)
const searchQuery = ref('')
const providerFilter = ref<Provider | ''>('')
const enabledFilter = ref<'' | 'true' | 'false'>('')
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })
const monitoringEnabled = computed(() => appStore.cachedPublicSettings?.channel_monitor_enabled !== false)

const showDialog = ref(false)
const showTemplateManager = ref(false)
const editing = ref<ChannelMonitor | null>(null)
const showDeleteDialog = ref(false)
const deleting = ref<ChannelMonitor | null>(null)
const showRunResult = ref(false)
const runResults = ref<CheckResult[]>([])
const showHistoryDialog = ref(false)
const historyTarget = ref<ChannelMonitor | null>(null)
const historyLoading = ref(false)
const historyEntries = ref<HistoryItem[]>([])
const historyModelFilter = ref('')

let abortController: AbortController | null = null
let historyAbortController: AbortController | null = null
let searchTimeout: ReturnType<typeof setTimeout> | null = null
const autoRefreshStorageKey = 'admin-channel-monitor-auto-refresh'

const columns = computed<Column[]>(() => [
  { key: 'name', label: t('admin.channelMonitor.columns.name'), sortable: false },
  { key: 'provider', label: t('admin.channelMonitor.columns.provider'), sortable: false },
  { key: 'primary_model', label: t('admin.channelMonitor.columns.primaryModel'), sortable: false },
  { key: 'availability_7d', label: t('admin.channelMonitor.columns.availability7d'), sortable: false },
  { key: 'latest_check', label: t('admin.channelMonitor.columns.latestCheck'), sortable: false },
  { key: 'latency', label: t('admin.channelMonitor.columns.latency'), sortable: false },
  { key: 'enabled', label: t('admin.channelMonitor.columns.enabled'), sortable: false },
  { key: 'actions', label: t('admin.channelMonitor.columns.actions'), sortable: false, class: 'min-w-[220px]' },
])

const deleteConfirmMessage = computed(() => {
  const name = deleting.value?.name || ''
  return t('admin.channelMonitor.deleteConfirm', { name })
})

const historyDialogTitle = computed(() => {
  const name = historyTarget.value?.name || ''
  return name
    ? t('admin.channelMonitor.historyTitleWithName', { name })
    : t('admin.channelMonitor.historyTitle')
})

const historyModelOptions = computed(() => {
  const target = historyTarget.value
  const models = target
    ? [target.primary_model, ...(target.extra_models || [])]
      .filter((model, index, list) => Boolean(model) && list.indexOf(model) === index)
    : []

  return [
    { value: '', label: t('admin.channelMonitor.historyAllModels') },
    ...models.map(model => ({ value: model, label: model })),
  ]
})

const autoRefresh = useAutoRefresh({
  storageKey: autoRefreshStorageKey,
  intervals: [15, 30, 60, 120] as const,
  defaultInterval: 30,
  onRefresh: () => reload(true),
  shouldPause: () => document.hidden
    || loading.value
    || showDialog.value
    || showDeleteDialog.value
    || showHistoryDialog.value
    || showTemplateManager.value,
})

function formatLastCheckedAt(value: string | null | undefined): string {
  if (!value) return t('common.time.never')
  return formatDateTime(value)
}

function hasSavedAutoRefreshPreference(): boolean {
  if (typeof window === 'undefined') return false
  try {
    return window.localStorage.getItem(autoRefreshStorageKey) !== null
  } catch {
    return false
  }
}

async function reload(silent = false) {
  if (abortController) abortController.abort()
  const ctrl = new AbortController()
  abortController = ctrl
  if (!silent) loading.value = true
  try {
    const params: ListParams = {
      page: pagination.page,
      page_size: pagination.page_size,
    }
    if (providerFilter.value) params.provider = providerFilter.value
    if (enabledFilter.value === 'true') params.enabled = true
    if (enabledFilter.value === 'false') params.enabled = false
    if (searchQuery.value.trim()) params.search = searchQuery.value.trim()

    const res = await adminAPI.channelMonitor.list(params, { signal: ctrl.signal })
    if (ctrl.signal.aborted || abortController !== ctrl) return
    monitors.value = res.items || []
    pagination.total = res.total
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.loadError')))
  } finally {
    if (abortController === ctrl) {
      if (!silent) loading.value = false
      autoRefresh.resetCountdown()
      abortController = null
    }
  }
}

function handleSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    reload()
  }, 300)
}

function onPageChange(page: number) {
  pagination.page = page
  reload()
}

function onPageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  reload()
}

function openHistoryDialog(row: ChannelMonitor) {
  historyTarget.value = row
  historyModelFilter.value = ''
  historyEntries.value = []
  showHistoryDialog.value = true
  void loadHistory()
}

function closeHistoryDialog() {
  showHistoryDialog.value = false
  historyTarget.value = null
  historyEntries.value = []
  historyModelFilter.value = ''
  historyAbortController?.abort()
}

async function loadHistory() {
  if (!historyTarget.value) return

  historyAbortController?.abort()
  const ctrl = new AbortController()
  historyAbortController = ctrl
  historyLoading.value = true

  try {
    const res = await adminAPI.channelMonitor.listHistory(historyTarget.value.id, {
      model: historyModelFilter.value || undefined,
      limit: 20,
    })
    if (ctrl.signal.aborted || historyAbortController !== ctrl) return
    historyEntries.value = res.items || []
  } catch (err: unknown) {
    const e = err as { name?: string; code?: string }
    if (e?.name === 'AbortError' || e?.code === 'ERR_CANCELED') return
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.historyLoadError')))
  } finally {
    if (historyAbortController === ctrl) {
      historyLoading.value = false
      historyAbortController = null
    }
  }
}

function openCreateDialog() {
  editing.value = null
  showDialog.value = true
}

function openEditDialog(row: ChannelMonitor) {
  editing.value = row
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  editing.value = null
}

async function toggleEnabled(row: ChannelMonitor) {
  const next = !row.enabled
  try {
    await adminAPI.channelMonitor.update(row.id, { enabled: next })
    row.enabled = next
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

async function handleRunNow(row: ChannelMonitor) {
  if (runningId.value != null) return
  runningId.value = row.id
  try {
    const res = await adminAPI.channelMonitor.runNow(row.id)
    runResults.value = res.results || []
    showRunResult.value = true
    appStore.showSuccess(t('admin.channelMonitor.runSuccess'))
    void reload()
    if (showHistoryDialog.value && historyTarget.value?.id === row.id) {
      void loadHistory()
    }
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.channelMonitor.runFailed')))
  } finally {
    runningId.value = null
  }
}

function handleDelete(row: ChannelMonitor) {
  deleting.value = row
  showDeleteDialog.value = true
}

async function confirmDelete() {
  if (!deleting.value) return
  try {
    await adminAPI.channelMonitor.del(deleting.value.id)
    appStore.showSuccess(t('admin.channelMonitor.deleteSuccess'))
    showDeleteDialog.value = false
    deleting.value = null
    reload()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('common.error')))
  }
}

watch(historyModelFilter, () => {
  if (showHistoryDialog.value && historyTarget.value) {
    void loadHistory()
  }
})

watch(
  () => monitoringEnabled.value,
  (enabled) => {
    if (!enabled) {
      autoRefresh.stop()
      return
    }
    if (autoRefresh.enabled.value) {
      autoRefresh.resetCountdown()
      autoRefresh.start()
    }
  },
)

onMounted(() => {
  void reload(false)
  if (!monitoringEnabled.value) return

  if (hasSavedAutoRefreshPreference()) {
    if (autoRefresh.enabled.value) {
      autoRefresh.resetCountdown()
      autoRefresh.start()
    }
    return
  }

  autoRefresh.setEnabled(true)
})

onUnmounted(() => {
  if (searchTimeout) clearTimeout(searchTimeout)
  abortController?.abort()
  historyAbortController?.abort()
  autoRefresh.stop()
})
</script>
