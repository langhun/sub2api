<template>
  <BaseDialog :show="show" :title="t('admin.openaiAt.title')" width="normal" @close="handleClose">
    <div v-if="account" class="space-y-4">
      <p class="rounded-lg bg-amber-50 p-3 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
        {{ t('admin.openaiAt.securityHint') }}
      </p>

      <div>
        <label class="input-label">{{ t('admin.openaiAt.oldToken') }}</label>
        <div class="flex gap-2">
          <input
            v-model="oldToken"
            :type="showOldToken ? 'text' : 'password'"
            readonly
            class="input min-w-0 flex-1 font-mono text-sm"
            autocomplete="off"
            data-testid="openai-at-old-token"
          />
          <button type="button" class="btn btn-secondary shrink-0" :disabled="!oldToken || loading" @click="toggleOldToken">
            {{ showOldToken ? t('admin.openaiAt.hide') : t('admin.openaiAt.reveal') }}
          </button>
          <button type="button" class="btn btn-secondary shrink-0" :disabled="!oldToken || loading" @click="copyOldToken">
            {{ copied ? t('admin.openaiAt.copied') : t('admin.openaiAt.copy') }}
          </button>
        </div>
        <p v-if="loading" class="input-hint">{{ t('admin.openaiAt.loading') }}</p>
      </div>

      <div>
        <label class="input-label">{{ t('admin.openaiAt.newToken') }}</label>
        <input
          v-model="newToken"
          type="password"
          class="input font-mono text-sm"
          autocomplete="new-password"
          :placeholder="t('admin.openaiAt.newTokenPlaceholder')"
          data-testid="openai-at-new-token"
          @keydown.enter.prevent="replaceToken"
        />
      </div>

      <p v-if="errorMessage" class="rounded-lg bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
        {{ errorMessage }}
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="loading || replacing || !newToken.trim()" @click="replaceToken">
          {{ replacing ? t('admin.openaiAt.replacing') : t('admin.openaiAt.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import type { Account } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { getOpenAIAT, replaceOpenAIAT } from './api'

defineOptions({ name: 'OpenAIATModal' })

const props = defineProps<{ show: boolean; account: Account | null }>()
const emit = defineEmits<{ close: []; updated: [account: Account] }>()
const { t } = useI18n()
const appStore = useAppStore()

const oldToken = ref('')
const newToken = ref('')
const showOldToken = ref(true)
const loading = ref(false)
const replacing = ref(false)
const copied = ref(false)
const errorMessage = ref('')

const clearSensitiveState = () => {
  oldToken.value = ''
  newToken.value = ''
  showOldToken.value = true
  copied.value = false
  errorMessage.value = ''
}

const loadToken = async () => {
  if (!props.show || !props.account) return
  clearSensitiveState()
  loading.value = true
  try {
    const result = await getOpenAIAT(props.account.id)
    oldToken.value = result.access_token
  } catch (error: any) {
    errorMessage.value = error?.response?.data?.message || error?.message || t('admin.openaiAt.loadFailed')
  } finally {
    loading.value = false
  }
}

const toggleOldToken = () => {
  showOldToken.value = !showOldToken.value
}

const copyOldToken = async () => {
  if (!oldToken.value) return
  try {
    await navigator.clipboard.writeText(oldToken.value)
    copied.value = true
    window.setTimeout(() => { copied.value = false }, 1500)
  } catch {
    errorMessage.value = t('admin.openaiAt.copyFailed')
  }
}

const replaceToken = async () => {
  if (!props.account) return
  const token = newToken.value.trim()
  if (!token) {
    errorMessage.value = t('admin.openaiAt.required')
    return
  }
  if (!window.confirm(t('admin.openaiAt.confirm'))) return

  replacing.value = true
  errorMessage.value = ''
  try {
    const result = await replaceOpenAIAT(props.account.id, token)
    appStore.showSuccess(t('admin.openaiAt.replaceSuccess'))
    emit('updated', result.account)
    handleClose()
  } catch (error: any) {
    errorMessage.value = error?.response?.data?.message || error?.message || t('admin.openaiAt.replaceFailed')
  } finally {
    replacing.value = false
  }
}

const handleClose = () => {
  clearSensitiveState()
  emit('close')
}

watch(() => [props.show, props.account?.id], loadToken, { immediate: true })
onBeforeUnmount(clearSensitiveState)
</script>
