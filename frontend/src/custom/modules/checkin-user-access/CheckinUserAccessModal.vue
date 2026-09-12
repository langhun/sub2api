<template>
  <BaseDialog :show="show" :title="t('admin.checkinUserAccess.title')" width="normal" @close="close">
    <div v-if="user" class="space-y-4">
      <div class="rounded-lg border border-gray-200 bg-gray-50 p-3 text-sm dark:border-dark-700 dark:bg-dark-800">
        <p class="text-gray-500 dark:text-dark-400">{{ t('admin.checkinUserAccess.user') }}</p>
        <p class="mt-1 font-medium text-gray-900 dark:text-white">{{ displayName }}</p>
      </div>

      <div v-if="loading" class="py-5 text-center text-sm text-gray-500 dark:text-dark-400">
        {{ t('common.loading') }}
      </div>

      <template v-else>
        <div
          class="rounded-lg border p-4"
          :class="access?.disabled
            ? 'border-amber-200 bg-amber-50 dark:border-amber-900/60 dark:bg-amber-900/20'
            : 'border-emerald-200 bg-emerald-50 dark:border-emerald-900/60 dark:bg-emerald-900/20'"
        >
          <p class="font-medium" :class="access?.disabled ? 'text-amber-800 dark:text-amber-200' : 'text-emerald-800 dark:text-emerald-200'">
            {{ access?.disabled ? t('admin.checkinUserAccess.disabled') : t('admin.checkinUserAccess.enabled') }}
          </p>
          <p class="mt-1 text-sm" :class="access?.disabled ? 'text-amber-700 dark:text-amber-300' : 'text-emerald-700 dark:text-emerald-300'">
            {{ access?.disabled ? t('admin.checkinUserAccess.disabledDescription') : t('admin.checkinUserAccess.enabledDescription') }}
          </p>
        </div>

        <p class="rounded-lg bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">
          {{ t('admin.checkinUserAccess.scopeHint') }}
        </p>

        <p v-if="errorMessage" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-900/20 dark:text-red-300">
          {{ errorMessage }}
        </p>
      </template>
    </div>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="saving" @click="close">{{ t('common.cancel') }}</button>
        <button
          v-if="access?.disabled"
          type="button"
          class="btn btn-primary"
          :disabled="loading || saving"
          @click="save(false)"
        >
          {{ saving ? t('common.saving') : t('admin.checkinUserAccess.restore') }}
        </button>
        <button
          v-else
          type="button"
          class="btn btn-danger"
          :disabled="loading || saving"
          @click="save(true)"
        >
          {{ saving ? t('common.saving') : t('admin.checkinUserAccess.disable') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useAppStore } from '@/stores/app'
import type { AdminUser } from '@/types'
import { extractApiErrorMessage } from '@/utils/apiError'
import { checkinUserAccessAPI, type CheckinUserAccess } from './api'

defineOptions({ name: 'CheckinUserAccessModal' })

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits<{ close: []; updated: [] }>()
const { t } = useI18n()
const appStore = useAppStore()

const access = ref<CheckinUserAccess | null>(null)
const loading = ref(false)
const saving = ref(false)
const errorMessage = ref('')
const displayName = computed(() => props.user?.username?.trim() || props.user?.email || '')

async function load() {
  if (!props.show || !props.user) return
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await checkinUserAccessAPI.get(props.user.id)
    access.value = response.data
  } catch (error) {
    access.value = null
    errorMessage.value = extractApiErrorMessage(error, '') || t('admin.checkinUserAccess.loadFailed')
  } finally {
    loading.value = false
  }
}

async function save(disabled: boolean) {
  if (!props.user) return
  const confirmation = disabled
    ? t('admin.checkinUserAccess.disableConfirm')
    : t('admin.checkinUserAccess.restoreConfirm')
  if (!window.confirm(confirmation)) return

  saving.value = true
  errorMessage.value = ''
  try {
    const response = await checkinUserAccessAPI.setDisabled(props.user.id, disabled)
    access.value = response.data
    appStore.showSuccess(disabled ? t('admin.checkinUserAccess.disableSuccess') : t('admin.checkinUserAccess.restoreSuccess'))
    emit('updated')
  } catch (error) {
    errorMessage.value = extractApiErrorMessage(error, '') || t('admin.checkinUserAccess.saveFailed')
  } finally {
    saving.value = false
  }
}

function close() {
  if (saving.value) return
  access.value = null
  errorMessage.value = ''
  emit('close')
}

watch(() => [props.show, props.user?.id], load, { immediate: true })
</script>
