<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.batchTest.title')"
    width="wide"
    @close="handleClose"
  >
    <div class="space-y-4">
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div class="rounded-lg border border-gray-200 bg-white p-3 dark:border-dark-500 dark:bg-dark-700">
          <div class="text-xs font-medium text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.batchTest.total') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-gray-900 dark:text-white">{{ rows.length }}</div>
        </div>
        <div class="rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-500/30 dark:bg-green-500/10">
          <div class="text-xs font-medium text-green-700 dark:text-green-300">
            {{ t('admin.accounts.batchTest.success') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-green-700 dark:text-green-300">{{ successCount }}</div>
        </div>
        <div class="rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-500/30 dark:bg-red-500/10">
          <div class="text-xs font-medium text-red-700 dark:text-red-300">
            {{ t('admin.accounts.batchTest.failed') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-red-700 dark:text-red-300">{{ failedCount }}</div>
        </div>
        <div class="rounded-lg border border-blue-200 bg-blue-50 p-3 dark:border-blue-500/30 dark:bg-blue-500/10">
          <div class="text-xs font-medium text-blue-700 dark:text-blue-300">
            {{ t('admin.accounts.batchTest.progress') }}
          </div>
          <div class="mt-1 text-2xl font-semibold text-blue-700 dark:text-blue-300">
            {{ completedCount }}/{{ rows.length }}
          </div>
        </div>
      </div>

      <div class="h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-600">
        <div
          class="h-full rounded-full bg-primary-500 transition-all"
          :style="{ width: progressPercent + '%' }"
        ></div>
      </div>

      <div class="grid gap-4 sm:grid-cols-[minmax(0,1fr)_220px]">
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.batchTest.model') }}
          </label>
          <Select
            v-model="selectedModelId"
            :options="availableModels"
            :disabled="running || loadingModels || availableModels.length === 0"
            value-key="id"
            label-key="display_name"
            :placeholder="modelPlaceholder"
            :empty-text="modelEmptyText"
            :searchable="availableModels.length > 5"
          />
          <p v-if="modelLoadError" class="text-xs text-red-600 dark:text-red-300">
            {{ modelLoadError }}
          </p>
          <p
            v-else-if="!loadingModels && rows.length > 0 && availableModels.length === 0"
            class="text-xs text-amber-600 dark:text-amber-300"
          >
            {{ t('admin.accounts.batchTest.noCommonModels') }}
          </p>
        </div>

        <div class="space-y-1.5">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.batchTest.concurrency') }}
          </label>
          <input
            v-model.number="concurrencyLimit"
            type="number"
            min="1"
            :max="MAX_BATCH_TEST_CONCURRENCY"
            :disabled="running"
            class="input"
            @input="concurrencyLimit = normalizeConcurrencyLimit(concurrencyLimit)"
          />
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.accounts.batchTest.concurrencyHint', { max: MAX_BATCH_TEST_CONCURRENCY }) }}
          </p>
        </div>
      </div>

      <div class="grid gap-4 sm:grid-cols-[220px_minmax(0,1fr)]">
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.accounts.batchTest.resultFilterLabel') }}
          </label>
          <Select
            v-model="resultFilter"
            :options="resultFilterOptions"
            :disabled="running && !hasCompleted"
          />
        </div>

        <div class="flex flex-wrap items-end gap-2">
          <button
            v-for="option in quickFilterOptions"
            :key="option.value"
            type="button"
            :data-testid="`batch-test-filter-${option.value}`"
            class="inline-flex items-center rounded-full border px-3 py-1.5 text-xs font-medium transition-colors"
            :class="quickFilterButtonClass(option.value)"
            @click="resultFilter = option.value"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <div class="max-h-[420px] overflow-y-auto rounded-lg border border-gray-200 dark:border-dark-500">
        <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-500">
          <thead class="sticky top-0 bg-gray-50 dark:bg-dark-700">
            <tr>
              <th class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.account') }}
              </th>
              <th class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.platform') }}
              </th>
              <th class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.columns.status') }}
              </th>
              <th class="px-4 py-3 text-left font-medium text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.batchTest.result') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 bg-white dark:divide-dark-600 dark:bg-dark-800">
            <tr v-for="row in filteredRows" :key="row.id" data-testid="batch-test-row">
              <td class="px-4 py-3">
                <div class="font-medium text-gray-900 dark:text-white">{{ row.name }}</div>
                <div class="text-xs text-gray-500 dark:text-gray-400">ID {{ row.id }}</div>
              </td>
              <td class="px-4 py-3 text-gray-600 dark:text-gray-300">
                <div class="flex flex-wrap gap-1.5">
                  <span class="rounded bg-gray-100 px-2 py-0.5 text-xs uppercase text-gray-700 dark:bg-dark-600 dark:text-gray-300">
                    {{ row.platform || '-' }}
                  </span>
                  <span class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-700 dark:bg-dark-600 dark:text-gray-300">
                    {{ row.type || '-' }}
                  </span>
                </div>
              </td>
              <td class="px-4 py-3">
                <span :class="statusBadgeClass(row.status)" class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium">
                  <Icon
                    v-if="row.status === 'running'"
                    name="refresh"
                    size="xs"
                    class="animate-spin"
                    :stroke-width="2"
                  />
                  <Icon v-else-if="row.status === 'success'" name="check" size="xs" :stroke-width="2" />
                  <Icon v-else-if="row.status === 'failed'" name="x" size="xs" :stroke-width="2" />
                  <span>{{ statusLabel(row.status) }}</span>
                </span>
              </td>
              <td class="max-w-[360px] px-4 py-3 text-gray-600 dark:text-gray-300">
                <div class="truncate" :title="row.message">{{ row.message || '-' }}</div>
              </td>
            </tr>
            <tr v-if="filteredRows.length === 0">
              <td colspan="4" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.batchTest.emptyFiltered') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          @click="handleClose"
          class="btn btn-secondary"
        >
          {{ t('common.close') }}
        </button>
        <button
          v-if="running"
          @click="stopBatch"
          class="btn btn-warning"
        >
          {{ t('admin.accounts.batchTest.stop') }}
        </button>
        <button
          v-else
          @click="startBatch"
          :disabled="rows.length === 0 || loadingModels || !selectedModelId"
          class="btn btn-primary"
        >
          {{ hasCompleted ? t('admin.accounts.batchTest.retry') : t('admin.accounts.batchTest.start') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { ClaudeModel } from '@/types'

type BatchTestStatus = 'pending' | 'running' | 'success' | 'failed' | 'skipped'

interface BatchTestTarget {
  id: number
  name: string
  platform?: string
  type?: string
}

interface BatchTestRow extends BatchTestTarget {
  status: BatchTestStatus
  message: string
  resultCode: string
}

type ResultFilterValue = 'all' | 'success' | '401' | '429' | 'other_failed'

interface TestStreamEvent {
  type: string
  text?: string
  model?: string
  success?: boolean
  error?: string
}

const props = defineProps<{
  show: boolean
  targets: BatchTestTarget[]
  defaultModelOnly?: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'completed', result: {
    success: number
    failed: number
    successIds: number[]
    failedIds: number[]
    unauthorizedFailedIds: number[]
  }): void
  (e: 'queue-delete', accountId: number): void
}>()

const { t } = useI18n()

const rows = ref<BatchTestRow[]>([])
const running = ref(false)
const stopRequested = ref(false)
const availableModels = ref<ClaudeModel[]>([])
const selectedModelId = ref('')
const resultFilter = ref<ResultFilterValue>('all')
const MAX_BATCH_TEST_CONCURRENCY = 50
const DEFAULT_BATCH_TEST_CONCURRENCY = 5
const concurrencyLimit = ref(DEFAULT_BATCH_TEST_CONCURRENCY)
const loadingModels = ref(false)
const modelLoadError = ref('')
const activeAbortControllers = new Set<AbortController>()
let activeRunToken = 0
let modelLoadSeq = 0
const DEFAULT_BATCH_MODEL_ID = '__default__'
const queuedDeleteIds = new Set<number>()

const completedCount = computed(() => rows.value.filter(row => ['success', 'failed', 'skipped'].includes(row.status)).length)
const successCount = computed(() => rows.value.filter(row => row.status === 'success').length)
const failedCount = computed(() => rows.value.filter(row => row.status === 'failed').length)
const statusCodeCounts = computed<Record<ResultFilterValue, number>>(() => {
  const counts: Record<ResultFilterValue, number> = {
    all: rows.value.length,
    success: 0,
    '401': 0,
    '429': 0,
    other_failed: 0
  }

  for (const row of rows.value) {
    if (row.status === 'success') {
      counts.success++
      continue
    }
    if (row.status !== 'failed') continue
    if (row.resultCode === '401') {
      counts['401']++
    } else if (row.resultCode === '429') {
      counts['429']++
    } else {
      counts.other_failed++
    }
  }

  return counts
})
const progressPercent = computed(() => {
  if (rows.value.length === 0) return 0
  return Math.round((completedCount.value / rows.value.length) * 100)
})
const hasCompleted = computed(() => completedCount.value > 0)
const modelPlaceholder = computed(() => loadingModels.value ? t('common.loading') : t('admin.accounts.batchTest.selectModel'))
const modelEmptyText = computed(() => loadingModels.value ? t('common.loading') : t('admin.accounts.batchTest.noCommonModels'))
const effectiveConcurrency = computed(() =>
  Math.min(rows.value.length || 1, normalizeConcurrencyLimit(concurrencyLimit.value))
)
const resultFilterOptions = computed<Array<{ value: ResultFilterValue; label: string }>>(() => [
  { value: 'all', label: t('admin.accounts.batchTest.resultFilters.all', { count: statusCodeCounts.value.all }) },
  { value: 'success', label: t('admin.accounts.batchTest.resultFilters.success', { count: statusCodeCounts.value.success }) },
  { value: '401', label: t('admin.accounts.batchTest.resultFilters.unauthorized', { count: statusCodeCounts.value['401'] }) },
  { value: '429', label: t('admin.accounts.batchTest.resultFilters.rateLimited', { count: statusCodeCounts.value['429'] }) },
  { value: 'other_failed', label: t('admin.accounts.batchTest.resultFilters.otherFailed', { count: statusCodeCounts.value.other_failed }) }
])
const quickFilterOptions = computed(() => resultFilterOptions.value)
const filteredRows = computed(() => {
  switch (resultFilter.value) {
    case 'success':
      return rows.value.filter(row => row.status === 'success')
    case '401':
      return rows.value.filter(row => row.status === 'failed' && row.resultCode === '401')
    case '429':
      return rows.value.filter(row => row.status === 'failed' && row.resultCode === '429')
    case 'other_failed':
      return rows.value.filter(
        row => row.status === 'failed' && row.resultCode !== '401' && row.resultCode !== '429'
      )
    default:
      return rows.value
  }
})

watch(
  () => props.show,
  (visible) => {
    if (visible) {
      resetRows()
      loadBatchModels()
    } else {
      stopBatch()
      resetModelState()
    }
  }
)

onMounted(() => {
  if (!props.show) return
  resetRows()
  loadBatchModels()
})

watch(
  () => props.targets.map(target => `${target.id}:${target.platform}:${target.type}`).join('|'),
  () => {
    if (!props.show) return
    resetRows()
    loadBatchModels()
  }
)

const resetRows = () => {
  queuedDeleteIds.clear()
  rows.value = props.targets.map(target => ({
    ...target,
    status: 'pending',
    message: '',
    resultCode: ''
  }))
  resultFilter.value = 'all'
}

const resetModelState = () => {
  modelLoadSeq++
  availableModels.value = []
  selectedModelId.value = ''
  concurrencyLimit.value = DEFAULT_BATCH_TEST_CONCURRENCY
  loadingModels.value = false
  modelLoadError.value = ''
}

const normalizeConcurrencyLimit = (value: number) => {
  if (!Number.isFinite(value)) return DEFAULT_BATCH_TEST_CONCURRENCY
  const normalized = Math.trunc(value)
  if (normalized < 1) return 1
  if (normalized > MAX_BATCH_TEST_CONCURRENCY) return MAX_BATCH_TEST_CONCURRENCY
  return normalized
}

const mergeCommonModels = (modelLists: ClaudeModel[][]): ClaudeModel[] => {
  if (modelLists.length === 0) return []
  const modelMaps = modelLists.map(models => new Map(models.map(model => [model.id, model])))
  const seen = new Set<string>()

  return modelLists[0].filter((model) => {
    if (!model.id || seen.has(model.id)) return false
    seen.add(model.id)
    return modelMaps.every(modelMap => modelMap.has(model.id))
  }).map((model) => ({
    ...model,
    display_name: model.display_name || model.id
  }))
}

const loadBatchModels = async () => {
  if (props.defaultModelOnly) {
    availableModels.value = [
      {
        id: DEFAULT_BATCH_MODEL_ID,
        display_name: t('admin.accounts.batchTest.defaultModelOption')
      } as ClaudeModel
    ]
    selectedModelId.value = DEFAULT_BATCH_MODEL_ID
    modelLoadError.value = ''
    loadingModels.value = false
    return
  }

  const targets = [...props.targets]
  selectedModelId.value = ''
  availableModels.value = []
  modelLoadError.value = ''

  if (targets.length === 0) return

  const seq = ++modelLoadSeq
  loadingModels.value = true
  try {
    const modelLists = await Promise.all(targets.map(target => adminAPI.accounts.getAvailableModels(target.id)))
    if (seq !== modelLoadSeq) return
    availableModels.value = mergeCommonModels(modelLists)
  } catch (error) {
    if (seq !== modelLoadSeq) return
    console.error('Failed to load batch test models:', error)
    availableModels.value = []
    modelLoadError.value = t('admin.accounts.batchTest.modelLoadFailed')
  } finally {
    if (seq === modelLoadSeq) {
      loadingModels.value = false
    }
  }
}

const setRow = (index: number, patch: Partial<BatchTestRow>) => {
  const current = rows.value[index]
  if (!current) return
  rows.value[index] = { ...current, ...patch }
}

const queueDeleteIfUnauthorized = (row: BatchTestRow, resultCode: string) => {
  if (resultCode !== '401' || queuedDeleteIds.has(row.id)) return
  queuedDeleteIds.add(row.id)
  emit('queue-delete', row.id)
}

const quickFilterButtonClass = (value: ResultFilterValue) => {
  const selected = resultFilter.value === value
  return selected
    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/20 dark:text-primary-200'
    : 'border-gray-200 bg-white text-gray-600 hover:border-primary-300 hover:text-primary-700 dark:border-dark-500 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:text-primary-200'
}

const statusLabel = (status: BatchTestStatus) => {
  return t(`admin.accounts.batchTest.status.${status}`)
}

const statusBadgeClass = (status: BatchTestStatus) => {
  switch (status) {
    case 'running':
      return 'bg-blue-100 text-blue-700 dark:bg-blue-500/20 dark:text-blue-300'
    case 'success':
      return 'bg-green-100 text-green-700 dark:bg-green-500/20 dark:text-green-300'
    case 'failed':
      return 'bg-red-100 text-red-700 dark:bg-red-500/20 dark:text-red-300'
    case 'skipped':
      return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-500/20 dark:text-yellow-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-600 dark:text-gray-300'
  }
}

const stopBatch = () => {
  activeRunToken++
  stopRequested.value = true
  for (const controller of activeAbortControllers) {
    controller.abort()
  }
  activeAbortControllers.clear()
  rows.value = rows.value.map((row) => {
    if (row.status === 'pending' || row.status === 'running') {
      return {
        ...row,
        status: 'skipped',
        message: t('admin.accounts.batchTest.stopped'),
        resultCode: ''
      }
    }
    return row
  })
  running.value = false
}

const handleClose = () => {
  stopBatch()
  emit('close')
}

const startBatch = async () => {
  if (running.value || rows.value.length === 0 || !selectedModelId.value) return

  const modelId = selectedModelId.value === DEFAULT_BATCH_MODEL_ID ? '' : selectedModelId.value
  const runToken = ++activeRunToken
  resetRows()
  running.value = true
  stopRequested.value = false

  let nextIndex = 0
  const workerCount = effectiveConcurrency.value
  const workers = Array.from({ length: workerCount }, async () => {
    while (true) {
      if (runToken !== activeRunToken) return
      const index = nextIndex++
      if (index >= rows.value.length) return
      if (stopRequested.value) return

      const row = rows.value[index]
      setRow(index, { status: 'running', message: t('admin.accounts.batchTest.testing') })
      const controller = new AbortController()
      activeAbortControllers.add(controller)

      try {
        const result = await testAccount(row.id, modelId, controller.signal)
        if (runToken !== activeRunToken) return
        setRow(index, {
          status: result.success ? 'success' : 'failed',
          message: result.message,
          resultCode: result.resultCode
        })
        if (!result.success) {
          queueDeleteIfUnauthorized(row, result.resultCode)
        }
      } catch (error) {
        if (runToken !== activeRunToken) return
        if (error instanceof DOMException && error.name === 'AbortError') {
          setRow(index, { status: 'skipped', message: t('admin.accounts.batchTest.stopped'), resultCode: '' })
          continue
        }
        const errorMessage = error instanceof Error ? error.message : t('common.error')
        const resultCode = classifyResultCode(errorMessage)
        setRow(index, {
          status: 'failed',
          message: errorMessage,
          resultCode
        })
        queueDeleteIfUnauthorized(row, resultCode)
      } finally {
        activeAbortControllers.delete(controller)
      }
    }
  })

  await Promise.all(workers)

  if (runToken !== activeRunToken) {
    return
  }

  running.value = false
  if (!stopRequested.value) {
    emit('completed', {
      success: successCount.value,
      failed: failedCount.value,
      successIds: rows.value.filter(row => row.status === 'success').map(row => row.id),
      failedIds: rows.value.filter(row => row.status === 'failed').map(row => row.id),
      unauthorizedFailedIds: rows.value
        .filter(row => row.status === 'failed' && row.resultCode === '401')
        .map(row => row.id)
    })
  }
}

const testAccount = async (
  accountId: number,
  modelId: string,
  signal: AbortSignal
): Promise<{ success: boolean; message: string; resultCode: string }> => {
  const response = await fetch(`/api/v1/admin/accounts/${accountId}/test`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${localStorage.getItem('auth_token')}`,
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({ model_id: modelId }),
    signal
  })

  if (!response.ok) {
    throw new Error(`HTTP ${response.status}`)
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error(t('admin.accounts.batchTest.noResponseBody'))
  }

  const decoder = new TextDecoder()
  let buffer = ''
  let completed = false
  let success = false
  let message = ''
  let responseText = ''
  let model = ''
  let resultCode = ''
  const handleStreamLine = (rawLine: string) => {
    const line = rawLine.trim()
    if (!line.startsWith('data:')) return

    const jsonStr = line.replace(/^data:\s*/, '').trim()
    if (!jsonStr) return

    const event = JSON.parse(jsonStr) as TestStreamEvent
    if (event.type === 'test_start') {
      model = event.model || model
    } else if (event.type === 'content' && event.text) {
      responseText += event.text
    } else if (event.type === 'test_complete') {
      completed = true
      success = Boolean(event.success)
      message = event.success
        ? formatSuccessMessage(model, responseText)
        : event.error || t('admin.accounts.batchTest.failedMessage')
      if (!event.success) {
        resultCode = classifyResultCode(message)
      }
    } else if (event.type === 'error') {
      completed = true
      success = false
      message = event.error || t('admin.accounts.batchTest.failedMessage')
      resultCode = classifyResultCode(message)
    }
  }

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const lines = buffer.split('\n')
    buffer = lines.pop() || ''

    for (const rawLine of lines) {
      handleStreamLine(rawLine)
    }
  }
  if (buffer.trim()) {
    handleStreamLine(buffer)
  }

  if (!completed) {
    return {
      success: false,
      message: t('admin.accounts.batchTest.incompleteStream'),
      resultCode: 'other'
    }
  }

  return { success, message, resultCode: success ? 'success' : resultCode || 'other' }
}

const formatSuccessMessage = (model: string, responseText: string) => {
  const trimmed = responseText.replace(/\s+/g, ' ').trim()
  if (model && trimmed) {
    return t('admin.accounts.batchTest.successWithModelAndText', {
      model,
      text: trimmed.slice(0, 80)
    })
  }
  if (model) {
    return t('admin.accounts.batchTest.successWithModel', { model })
  }
  return t('admin.accounts.batchTest.successMessage')
}

const classifyResultCode = (message: string) => {
  const normalized = message.trim()
  if (/(^|\D)401(\D|$)/.test(normalized) || /unauthorized/i.test(normalized)) {
    return '401'
  }
  if (/(^|\D)429(\D|$)/.test(normalized) || /rate\s*limit/i.test(normalized)) {
    return '429'
  }
  return 'other'
}
</script>
