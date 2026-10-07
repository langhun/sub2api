<template>
  <BaseDialog :show="show" :title="t('admin.openaiRt.title')" width="normal" @close="handleClose">
    <div v-if="account" class="space-y-4">
      <p class="rounded-xl bg-amber-50 p-3 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
        {{ t('admin.openaiRt.securityHint') }}
      </p>

      <div>
        <label class="input-label">{{ t('admin.openaiRt.oldToken') }}</label>
        <div class="flex gap-2">
          <input
            v-model="oldToken"
            :type="showOldToken ? 'text' : 'password'"
            readonly
            class="input min-w-0 flex-1 font-mono text-sm"
            autocomplete="off"
            data-testid="openai-rt-old-token"
          />
          <button type="button" class="btn btn-secondary shrink-0" :disabled="!oldToken || loading" @click="toggleOldToken">
            {{ showOldToken ? t('admin.openaiRt.hide') : t('admin.openaiRt.reveal') }}
          </button>
          <button type="button" class="btn btn-secondary shrink-0" :disabled="!oldToken || loading" @click="copyOldToken">
            {{ copied ? t('admin.openaiRt.copied') : t('admin.openaiRt.copy') }}
          </button>
        </div>
        <p v-if="loading" class="input-hint">{{ t('admin.openaiRt.loading') }}</p>
      </div>

      <div>
        <label class="input-label">{{ t('admin.openaiRt.newToken') }}</label>
        <input
          v-model="newToken"
          type="password"
          class="input font-mono text-sm"
          autocomplete="new-password"
          :placeholder="t('admin.openaiRt.newTokenPlaceholder')"
          data-testid="openai-rt-new-token"
          @keydown.enter.prevent="replaceToken"
        />
      </div>

      <p v-if="errorMessage" class="rounded-xl bg-red-50 p-3 text-sm text-red-600 dark:bg-red-900/20 dark:text-red-300">
        {{ errorMessage }}
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="handleClose">{{ t('common.cancel') }}</button>
        <button type="button" class="btn btn-primary" :disabled="loading || replacing || !newToken.trim()" @click="replaceToken">
          {{ replacing ? t('admin.openaiRt.replacing') : t('admin.openaiRt.save') }}
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
import { getOpenAIRT, replaceOpenAIRT } from './api'
import { extractApiErrorMessage } from '@/utils/apiError'

defineOptions({ name: 'OpenAIRTModal' })

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
    const result = await getOpenAIRT(props.account.id)
    oldToken.value = result.refresh_token
  } catch (error: any) {
    errorMessage.value = extractApiErrorMessage(error, '') || t('admin.openaiRt.loadFailed')
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
    errorMessage.value = t('admin.openaiRt.copyFailed')
  }
}

const replaceToken = async () => {
  if (!props.account) return
  const token = newToken.value.trim()
  if (!token) {
    errorMessage.value = t('admin.openaiRt.required')
    return
  }
  if (!window.confirm(t('admin.openaiRt.confirm'))) return

  replacing.value = true
  errorMessage.value = ''
  try {
    const result = await replaceOpenAIRT(props.account.id, token)
    appStore.showSuccess(t('admin.openaiRt.replaceSuccess'))
    emit('updated', result.account)
    handleClose()
  } catch (error: any) {
    const reason = error?.reason || error?.response?.data?.reason
    errorMessage.value = reason === 'OPENAI_RT_INVALID'
      ? t('admin.openaiRt.invalidToken')
      : extractApiErrorMessage(error, '') || t('admin.openaiRt.replaceFailed')
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
