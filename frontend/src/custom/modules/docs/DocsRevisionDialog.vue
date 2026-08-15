<template>
  <BaseDialog :show="show" :title="title" width="full" :close-on-click-outside="false" @close="emit('close')">
    <div class="docs-editor-grid">
      <section class="docs-editor-pane">
        <label class="docs-editor-label" for="docs-edit-summary">{{ t('docs.editSummary') }}</label>
        <input id="docs-edit-summary" v-model="editSummary" class="docs-editor-summary" :placeholder="t('docs.editSummaryPlaceholder')" maxlength="200" />
        <label class="docs-editor-label" for="docs-editor-content">Markdown</label>
        <textarea id="docs-editor-content" v-model="content" class="docs-editor-textarea" :placeholder="t('docs.editorPlaceholder')" spellcheck="false" />
        <p class="docs-editor-hint">{{ t('docs.markdownHint') }}</p>
      </section>
      <section class="docs-editor-pane">
        <h4 class="docs-editor-label">{{ t('docs.preview') }}</h4>
        <article class="docs-editor-preview docs-markdown" v-html="previewHtml" />
      </section>
    </div>
    <template #footer>
      <button type="button" class="docs-dialog-button" @click="emit('close')">{{ t('docs.cancel') }}</button>
      <button type="button" class="docs-dialog-button docs-dialog-button-primary" :disabled="saving || !content.trim()" @click="submit">
        <Icon v-if="!saving" name="check" size="sm" />
        <span>{{ saving ? t('docs.saving') : t('docs.saveRevision') }}</span>
      </button>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  show: boolean
  title: string
  initialContent: string
  initialSummary: string
  saving: boolean
}>()

const emit = defineEmits<{
  (event: 'close'): void
  (event: 'save', payload: { content: string; editSummary: string }): void
}>()

const { t } = useI18n()
const content = ref('')
const editSummary = ref('')

watch(() => [props.show, props.initialContent, props.initialSummary] as const, ([show]) => {
  if (!show) return
  content.value = props.initialContent
  editSummary.value = props.initialSummary
}, { immediate: true })

const previewHtml = computed(() => {
  const html = marked.parse(content.value || '') as string
  return DOMPurify.sanitize(html, {
    FORBID_TAGS: ['script', 'style', 'iframe', 'object', 'embed', 'form', 'video', 'audio', 'source', 'picture'],
    FORBID_ATTR: ['style', 'srcset'],
  })
})

function submit() {
  emit('save', { content: content.value, editSummary: editSummary.value })
}
</script>

<style scoped>
.docs-editor-grid { @apply grid gap-5 lg:grid-cols-2; }
.docs-editor-pane { @apply min-w-0; }
.docs-editor-label { @apply mb-2 block text-sm font-semibold text-gray-800 dark:text-dark-100; }
.docs-editor-summary { @apply mb-4 w-full border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500 dark:border-dark-600 dark:bg-dark-800 dark:text-white; border-radius: 6px; }
.docs-editor-textarea { @apply min-h-[28rem] w-full resize-y border border-gray-300 bg-gray-950 p-4 font-mono text-sm leading-6 text-gray-100 outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500 dark:border-dark-600; border-radius: 6px; }
.docs-editor-hint { @apply mt-2 text-xs text-gray-500 dark:text-dark-400; }
.docs-editor-preview { @apply max-h-[38rem] min-h-[28rem] overflow-y-auto border border-gray-200 p-5 dark:border-dark-700; border-radius: 6px; }
.docs-dialog-button { @apply inline-flex items-center gap-2 border border-gray-300 px-3 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:text-dark-200 dark:hover:bg-dark-800; border-radius: 6px; }
.docs-dialog-button-primary { @apply border-primary-600 bg-primary-600 text-white hover:bg-primary-700 dark:border-primary-500 dark:bg-primary-500 dark:hover:bg-primary-600; }
</style>
