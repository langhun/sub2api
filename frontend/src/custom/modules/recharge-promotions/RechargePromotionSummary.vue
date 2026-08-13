<template>
  <div v-if="showSummary" class="rounded-lg border border-emerald-200 bg-emerald-50/70 px-4 py-3 text-sm dark:border-emerald-900/70 dark:bg-emerald-950/20">
    <p class="font-medium text-emerald-800 dark:text-emerald-200">{{ t('rechargePromotion.condition') }}</p>
    <div v-if="loading" class="mt-2 text-xs text-emerald-700/80 dark:text-emerald-300/80">{{ t('common.loading') }}</div>
    <template v-else-if="preview?.active">
      <div class="mt-3 space-y-2">
        <div class="flex justify-between gap-4 text-gray-700 dark:text-gray-200">
          <span>{{ t('rechargePromotion.baseCredit') }}</span>
          <span class="font-medium tabular-nums">${{ preview.base_credit.toFixed(2) }}</span>
        </div>
        <div class="flex justify-between gap-4 text-emerald-800 dark:text-emerald-200">
          <span>{{ t('rechargePromotion.bonusCredit') }}</span>
          <span class="font-semibold tabular-nums">+${{ preview.bonus_credit.toFixed(2) }}</span>
        </div>
        <div class="flex justify-between gap-4 border-t border-emerald-200 pt-2 font-semibold text-emerald-900 dark:border-emerald-900/70 dark:text-emerald-100">
          <span>{{ t('rechargePromotion.finalCredit') }}</span>
          <span class="tabular-nums">${{ preview.final_credit.toFixed(2) }}</span>
        </div>
      </div>
    </template>
    <p v-else class="mt-2 text-xs text-emerald-800/90 dark:text-emerald-200/90">{{ inactiveMessage }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { rechargePromotionsAPI, type RechargePromotionPreview } from './api'

const props = defineProps<{ amount: number; baseCredit: number }>()
const { t } = useI18n()
const preview = ref<RechargePromotionPreview | null>(null)
const loading = ref(false)
let requestSequence = 0
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false

const previewDebounceMs = 250

const showSummary = computed(() => props.amount > 0 && props.baseCredit > 0)
const inactiveMessage = computed(() => {
  switch (preview.value?.ineligible_reason) {
    case 'disabled': return t('rechargePromotion.disabled')
    case 'not_started': return t('rechargePromotion.notStarted')
    case 'ended': return t('rechargePromotion.ended')
    case 'minimum_amount': return t('rechargePromotion.minimumAmount', { amount: preview.value.minimum_amount.toFixed(2) })
    default: return t('rechargePromotion.inactive')
  }
})

function clearPendingPreview() {
  if (debounceTimer !== null) {
    clearTimeout(debounceTimer)
    debounceTimer = null
  }
}

function schedulePreview(amount: number, baseCredit: number) {
  const sequence = ++requestSequence
  clearPendingPreview()
  preview.value = null
  loading.value = false
  if (amount <= 0 || baseCredit <= 0) return

  loading.value = true
  debounceTimer = setTimeout(async () => {
    debounceTimer = null
    try {
      const response = await rechargePromotionsAPI.preview(amount, baseCredit)
      if (!disposed && sequence === requestSequence) preview.value = response.data
    } catch {
      // Preview is advisory. A failed preview must never block upstream checkout.
    } finally {
      if (!disposed && sequence === requestSequence) loading.value = false
    }
  }, previewDebounceMs)
}

watch(() => [props.amount, props.baseCredit] as const, ([amount, baseCredit]) => {
  schedulePreview(amount, baseCredit)
}, { immediate: true })

onBeforeUnmount(() => {
  disposed = true
  requestSequence++
  clearPendingPreview()
})
</script>
