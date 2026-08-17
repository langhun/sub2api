<template>
  <BaseDialog
    :show="show"
    :title="t('checkin.verificationTitle')"
    width="narrow"
    :close-on-click-outside="!checkinStore.loading"
    :close-on-escape="!checkinStore.loading"
    @close="closeDialog"
  >
    <p class="mb-4 text-sm text-gray-600 dark:text-dark-300">
      {{ t('checkin.verificationDescription') }}
    </p>
    <CheckinCaptchaGate
      ref="captchaRef"
      :active="show"
      @ready-change="captchaReady = $event"
    />
    <div
      v-if="checkinStore.actionError"
      class="mt-3 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-900/20 dark:text-red-300"
      role="alert"
    >
      {{ actionErrorMessage }}
    </div>

    <template #footer>
      <div class="flex items-center justify-end gap-3">
        <button type="button" class="btn btn-secondary" :disabled="checkinStore.loading" @click="closeDialog">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          data-testid="normal-checkin-confirm"
          class="btn btn-primary"
          :disabled="checkinStore.loading || !captchaReady"
          @click="submit"
        >
          {{ checkinStore.loading ? t('common.loading') : t('checkin.verificationConfirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { CheckinResult } from '../api/checkin'
import { useCheckinStore } from '../stores/checkin'
import CheckinCaptchaGate from './CheckinCaptchaGate.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: []; success: [result: CheckinResult] }>()

const { t } = useI18n()
const checkinStore = useCheckinStore()
const captchaRef = ref<InstanceType<typeof CheckinCaptchaGate> | null>(null)
const captchaReady = ref(false)
const actionErrorMessage = computed(() => extractApiErrorMessage(checkinStore.actionError, t('checkin.actionFailed')))

watch(() => props.show, (show) => {
  if (!show) return
  checkinStore.clearActionError()
})

async function submit(): Promise<void> {
  if (checkinStore.loading || !captchaReady.value) return
  const proof = await captchaRef.value?.resolveProof()
  if (!proof) return
  const result = await checkinStore.doCheckin(proof)
  captchaRef.value?.reset()
  if (!result) return
  emit('success', result)
  emit('close')
}

function closeDialog(): void {
  if (checkinStore.loading) return
  captchaRef.value?.reset()
  checkinStore.clearActionError()
  emit('close')
}
</script>
