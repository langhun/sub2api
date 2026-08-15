<template>
  <BaseDialog :show="show" :title="title" width="wide" @close="emit('close')">
    <div v-if="history.length" class="docs-history-list">
      <article v-for="revision in history" :key="revision.id" class="docs-history-item" :class="{ 'docs-history-current': revision.id === currentRevisionId }">
        <div class="docs-history-meta">
          <div>
            <strong>#{{ revision.id }}</strong>
            <span>{{ formatDate(revision.created_at) }}</span>
            <span>{{ t('docs.revisionBy', { author: revision.author_name }) }}</span>
            <span v-if="revision.id === currentRevisionId" class="docs-history-badge">{{ t('docs.current') }}</span>
            <span :class="['docs-history-badge', `docs-history-status-${revision.status || 'approved'}`]">{{ statusLabel(revision.status) }}</span>
          </div>
          <div class="docs-history-actions">
            <template v-if="canReview && revision.status === 'pending'">
              <button type="button" class="docs-history-action docs-history-approve" :disabled="reviewingId === revision.id" @click="emit('review', revision, 'approve')">
                <Icon name="check" size="sm" />
                {{ t('docs.approve') }}
              </button>
              <button type="button" class="docs-history-action" :disabled="reviewingId === revision.id" @click="emit('review', revision, 'reject')">
                <Icon name="x" size="sm" />
                {{ t('docs.reject') }}
              </button>
            </template>
            <button v-if="canRevert && revision.id !== currentRevisionId && revision.status === 'approved'" type="button" class="docs-history-action" :disabled="revertingId === revision.id" @click="emit('revert', revision)">
              <Icon name="refresh" size="sm" />
              {{ t('docs.revert') }}
            </button>
          </div>
        </div>
        <p v-if="revision.edit_summary" class="docs-history-summary">{{ revision.edit_summary }}</p>
        <details class="docs-history-details">
          <summary>{{ t('docs.viewRevision') }}</summary>
          <pre>{{ revision.content }}</pre>
        </details>
      </article>
    </div>
    <p v-else class="docs-history-empty">{{ t('docs.noHistory') }}</p>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import type { DocsRevision } from './api'

defineProps<{
  show: boolean
  title: string
  history: DocsRevision[]
  currentRevisionId: number | null
  canRevert: boolean
  revertingId: number | null
  canReview: boolean
  reviewingId: number | null
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'revert', revision: DocsRevision): void
  (event: 'review', revision: DocsRevision, decision: 'approve' | 'reject'): void
}>()

const { t } = useI18n()

function formatDate(value: string) {
  return new Date(value).toLocaleString()
}

function statusLabel(status: DocsRevision['status']) {
  if (status === 'pending') return t('docs.pending')
  if (status === 'rejected') return t('docs.rejectedStatus')
  return t('docs.approved')
}
</script>

<style scoped>
.docs-history-list { @apply space-y-3; }
.docs-history-item { @apply border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60; border-radius: 6px; }
.docs-history-current { @apply border-primary-300 dark:border-primary-700; }
.docs-history-meta { @apply flex flex-wrap items-start justify-between gap-3 text-sm text-gray-600 dark:text-dark-300; }
.docs-history-meta > div { @apply flex flex-wrap items-center gap-2; }
.docs-history-actions { @apply flex flex-wrap items-center gap-2; }
.docs-history-meta strong { @apply font-mono text-gray-900 dark:text-white; }
.docs-history-badge { @apply bg-primary-100 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300; border-radius: 4px; }
.docs-history-status-pending { @apply bg-amber-100 text-amber-800 dark:bg-amber-900/30 dark:text-amber-300; }
.docs-history-status-rejected { @apply bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300; }
.docs-history-status-approved { @apply bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300; }
.docs-history-action { @apply inline-flex items-center gap-1.5 border border-gray-300 px-2.5 py-1.5 text-xs font-medium text-gray-700 hover:bg-white disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-dark-200 dark:hover:bg-dark-700; border-radius: 5px; }
.docs-history-approve { @apply border-green-300 text-green-700 hover:bg-green-50 dark:border-green-700 dark:text-green-300 dark:hover:bg-green-900/20; }
.docs-history-summary { @apply mt-3 text-sm text-gray-800 dark:text-dark-100; }
.docs-history-details { @apply mt-3 text-xs text-gray-500 dark:text-dark-400; }
.docs-history-details summary { @apply cursor-pointer select-none; }
.docs-history-details pre { @apply mt-2 max-h-64 overflow-auto whitespace-pre-wrap border border-gray-200 bg-white p-3 font-mono text-xs leading-5 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-200; border-radius: 5px; }
.docs-history-empty { @apply py-8 text-center text-sm text-gray-500 dark:text-dark-400; }
</style>
