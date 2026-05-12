<template>
  <div class="mb-4 flex items-center justify-between rounded-lg bg-primary-50 p-3 dark:bg-primary-900/20">
    <div class="flex flex-wrap items-center gap-2">
      <span v-if="selectedIds.length > 0" class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkActions.selected', { count: selectedIds.length }) }}
      </span>
      <span v-else class="text-sm font-medium text-primary-900 dark:text-primary-100">
        {{ t('admin.accounts.bulkEdit.title') }}
      </span>
      <template v-if="selectedIds.length > 0">
      <button
        @click="$emit('select-page')"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.selectCurrentPage') }}
      </button>
      <span class="text-gray-300 dark:text-primary-800">•</span>
      <button
        @click="$emit('clear')"
        class="text-xs font-medium text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200"
      >
        {{ t('admin.accounts.bulkActions.clear') }}
      </button>
      </template>
    </div>
    <div class="flex flex-wrap justify-end gap-2">
      <div
        v-if="showTestAllUngrouped"
        class="flex flex-wrap items-center justify-end gap-2 rounded-lg border border-primary-200/70 bg-white/70 px-2.5 py-1.5 dark:border-primary-700/40 dark:bg-primary-950/10"
      >
        <span class="text-xs font-medium text-primary-800 dark:text-primary-200">
          {{ t('admin.accounts.bulkActions.testUngroupedPrefix') }}
        </span>
        <input
          :value="ungroupedTestLimit"
          type="number"
          min="1"
          data-testid="account-bulk-test-all-ungrouped-limit"
          class="input h-8 w-20 px-2 py-1 text-sm"
          @input="handleUngroupedLimitInput"
        />
        <span class="text-xs text-primary-700/80 dark:text-primary-300/80">
          {{ t('admin.accounts.bulkActions.testUngroupedCountHint', { total: ungroupedTotalCount }) }}
        </span>
        <button
          data-testid="account-bulk-test-all-ungrouped"
          @click="$emit('test-all-ungrouped')"
          :disabled="testAllUngroupedLoading"
          class="btn btn-secondary btn-sm"
        >
          {{
            testAllUngroupedLoading
              ? t('admin.accounts.batchTest.loadingTargets')
              : t('admin.accounts.bulkActions.testAllUngrouped')
          }}
        </button>
      </div>
      <template v-if="selectedIds.length > 0">
        <button @click="$emit('delete')" class="btn btn-danger btn-sm">{{ t('admin.accounts.bulkActions.delete') }}</button>
        <button @click="$emit('test')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.test') }}</button>
        <button @click="$emit('reset-status')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.resetStatus') }}</button>
        <button @click="$emit('refresh-token')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.refreshToken') }}</button>
        <button data-testid="account-bulk-set-privacy" @click="$emit('set-privacy')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.setPrivacy') }}</button>
        <button data-testid="account-bulk-clear-privacy" @click="$emit('clear-privacy')" class="btn btn-secondary btn-sm">{{ t('admin.accounts.bulkActions.clearPrivacy') }}</button>
        <button @click="$emit('toggle-schedulable', true)" class="btn btn-success btn-sm">{{ t('admin.accounts.bulkActions.enableScheduling') }}</button>
        <button @click="$emit('toggle-schedulable', false)" class="btn btn-warning btn-sm">{{ t('admin.accounts.bulkActions.disableScheduling') }}</button>
        <button data-testid="account-bulk-edit-selected" @click="$emit('edit-selected')" class="btn btn-primary btn-sm">{{ t('admin.accounts.bulkActions.edit') }}</button>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{
  selectedIds: number[]
  showTestAllUngrouped?: boolean
  testAllUngroupedLoading?: boolean
  ungroupedTestLimit?: number
  ungroupedTotalCount?: number
}>()

const emit = defineEmits([
  'delete',
  'edit-selected',
  'clear',
  'select-page',
  'toggle-schedulable',
  'reset-status',
  'refresh-token',
  'set-privacy',
  'clear-privacy',
  'test',
  'test-all-ungrouped',
  'update:ungrouped-test-limit'
])

const { t } = useI18n()

const handleUngroupedLimitInput = (event: Event) => {
  const target = event.target as HTMLInputElement
  const raw = Number(target.value)
  if (!Number.isFinite(raw)) {
    return
  }
  const normalized = Math.max(1, Math.trunc(raw))
  target.value = String(normalized)
  emit('update:ungrouped-test-limit', normalized)
}
</script>
