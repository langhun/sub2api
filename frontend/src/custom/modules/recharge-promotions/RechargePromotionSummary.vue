<template>
  <div class="recharge-promotion text-sm" :aria-busy="loading">
    <div class="space-y-4">
      <div class="flex justify-between gap-4 text-emerald-800 dark:text-emerald-200">
        <span>{{ t('rechargePromotion.bonusCredit') }}</span>
        <span data-testid="promotion-bonus" class="font-semibold tabular-nums">{{ bonusText }}</span>
      </div>
      <div class="flex justify-between gap-4 font-semibold text-gray-900 dark:text-gray-100">
        <span>{{ t('rechargePromotion.finalCredit') }}</span>
        <span data-testid="promotion-final" class="tabular-nums">{{ finalText }}</span>
      </div>
    </div>
    <p class="promotion-status mt-2 text-xs leading-4 text-gray-500 dark:text-gray-400" role="status">{{ statusMessage }}</p>
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
const failed = ref(false)
let requestSequence = 0
let debounceTimer: ReturnType<typeof setTimeout> | null = null
let disposed = false

const previewDebounceMs = 250

const showSummary = computed(() => props.amount > 0 && props.baseCredit > 0)
// Keep the same rows while fetching; an unresolved bonus must not look confirmed.
const bonusText = computed(() => loading.value || failed.value ? '\u2014' : `+$${(preview.value?.active ? preview.value.bonus_credit : 0).toFixed(2)}`)
const finalText = computed(() => loading.value || failed.value ? '\u2014' : `$${(preview.value?.active ? preview.value.final_credit : props.baseCredit).toFixed(2)}`)
const statusMessage = computed(() => {
  if (!showSummary.value) return t('payment.enterAmount')
  if (loading.value) return t('common.loading')
  if (failed.value) return t('rechargePromotion.loadFailed')
  return preview.value?.active ? t('rechargePromotion.condition') : inactiveMessage.value
})
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
  failed.value = false
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
      if (!disposed && sequence === requestSequence) failed.value = true
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

<style scoped>
.promotion-status {
  min-height: 3rem;
  overflow-wrap: anywhere;
}
</style>
