<template>
  <section v-if="record" class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <h3 class="mb-3 text-sm font-semibold text-gray-800 dark:text-gray-200">{{ t('rechargePromotion.detail') }}</h3>
    <div class="grid grid-cols-2 gap-3 text-sm sm:grid-cols-3">
      <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('rechargePromotion.baseCredit') }}</p><p class="font-medium tabular-nums">${{ record.base_credit.toFixed(2) }}</p></div>
      <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('rechargePromotion.bonusCredit') }}</p><p class="font-medium tabular-nums text-emerald-700 dark:text-emerald-300">+${{ record.bonus_credit.toFixed(2) }}</p></div>
      <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('rechargePromotion.finalCredit') }}</p><p class="font-semibold tabular-nums">${{ record.final_credit.toFixed(2) }}</p></div>
      <template v-if="record.refunded_credit > 0">
        <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('rechargePromotion.refundedBase') }}</p><p class="font-medium tabular-nums text-red-600 dark:text-red-300">-${{ record.refunded_base_credit.toFixed(2) }}</p></div>
        <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('rechargePromotion.refundedBonus') }}</p><p class="font-medium tabular-nums text-red-600 dark:text-red-300">-${{ record.refunded_bonus_credit.toFixed(2) }}</p></div>
        <div><p class="text-xs text-gray-500 dark:text-gray-400">{{ t('rechargePromotion.refundedTotal') }}</p><p class="font-medium tabular-nums text-red-600 dark:text-red-300">-${{ record.refunded_credit.toFixed(2) }}</p></div>
      </template>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { rechargePromotionsAPI, type RechargePromotionOrder } from './api'

const props = defineProps<{ orderId: number; admin?: boolean }>()
const { t } = useI18n()
const record = ref<RechargePromotionOrder | null>(null)
let requestSequence = 0

watch(() => [props.orderId, props.admin] as const, async ([orderId, admin]) => {
  const sequence = ++requestSequence
  record.value = null
  if (!Number.isFinite(orderId) || orderId <= 0) return
  try {
    const response = admin
      ? await rechargePromotionsAPI.getAdminOrder(orderId)
      : await rechargePromotionsAPI.getUserOrder(orderId)
    if (sequence === requestSequence) record.value = response.data
  } catch {
    // A 404 simply means the order had no promotion snapshot.
  }
}, { immediate: true })
</script>
