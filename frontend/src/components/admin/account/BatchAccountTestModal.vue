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
            <tr v-for="row in rows" :key="row.id">
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
}

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
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'completed', result: { success: number; failed: number; successIds: number[]; failedIds: number[] }): void
}>()

const { t } = useI18n()

const rows = ref<BatchTestRow[]>([])
const running = ref(false)
const stopRequested = ref(false)
const availableModels = ref<ClaudeModel[]>([])
const selectedModelId = ref('')
const loadingModels = ref(false)
const modelLoadError = ref('')
let abortController: AbortController | null = null
let modelLoadSeq = 0

const completedCount = computed(() => rows.value.filter(row => ['success', 'failed', 'skipped'].includes(row.status)).length)
const successCount = computed(() => rows.value.filter(row => row.status === 'success').length)
const failedCount = computed(() => rows.value.filter(row => row.status === 'failed').length)
const progressPercent = computed(() => {
  if (rows.value.length === 0) return 0
  return Math.round((completedCount.value / rows.value.length) * 100)
})
const hasCompleted = computed(() => completedCount.value > 0)
const modelPlaceholder = computed(() => loadingModels.value ? t('common.loading') : t('admin.accounts.batchTest.selectModel'))
const modelEmptyText = computed(() => loadingModels.value ? t('common.loading') : t('admin.accounts.batchTest.noCommonModels'))

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
  rows.value = props.targets.map(target => ({
    ...target,
    status: 'pending',
    message: ''
  }))
}

const resetModelState = () => {
  modelLoadSeq++
  availableModels.value = []
  selectedModelId.value = ''
  loadingModels.value = false
  modelLoadError.value = ''
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
  stopRequested.value = true
  abortController?.abort()
  abortController = null
  running.value = false
}

const handleClose = () => {
  stopBatch()
  emit('close')
}

const startBatch = async () => {
  if (running.value || rows.value.length === 0 || !selectedModelId.value) return

  const modelId = selectedModelId.value
  resetRows()
  running.value = true
  stopRequested.value = false

  for (let index = 0; index < rows.value.length; index++) {
    if (stopRequested.value) {
      setRow(index, { status: 'skipped', message: t('admin.accounts.batchTest.stopped') })
      continue
    }

    const row = rows.value[index]
    setRow(index, { status: 'running', message: t('admin.accounts.batchTest.testing') })
    abortController = new AbortController()

    try {
      const result = await testAccount(row.id, modelId, abortController.signal)
      setRow(index, {
        status: result.success ? 'success' : 'failed',
        message: result.message
      })
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') {
        setRow(index, { status: 'skipped', message: t('admin.accounts.batchTest.stopped') })
        continue
      }
      setRow(index, {
        status: 'failed',
        message: error instanceof Error ? error.message : t('common.error')
      })
    } finally {
      abortController = null
    }
  }

  running.value = false
  if (!stopRequested.value) {
    emit('completed', {
      success: successCount.value,
      failed: failedCount.value,
      successIds: rows.value.filter(row => row.status === 'success').map(row => row.id),
      failedIds: rows.value.filter(row => row.status === 'failed').map(row => row.id)
    })
  }
}

const testAccount = async (accountId: number, modelId: string, signal: AbortSignal): Promise<{ success: boolean; message: string }> => {
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
    } else if (event.type === 'error') {
      completed = true
      success = false
      message = event.error || t('admin.accounts.batchTest.failedMessage')
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
      message: t('admin.accounts.batchTest.incompleteStream')
    }
  }

  return { success, message }
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
</script>
