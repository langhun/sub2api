<template>
  <template v-if="promotion.active">
    <div class="flex justify-between border-t border-gray-200 pt-2 dark:border-dark-600">
      <span class="text-gray-500 dark:text-gray-400">{{ localText('活动赠送', 'Promotion bonus') }}</span>
      <span class="font-medium text-emerald-700 dark:text-emerald-300">+${{ promotion.bonus.toFixed(2) }}</span>
    </div>
    <div class="flex justify-between">
      <span class="font-medium text-gray-700 dark:text-gray-300">{{ localText('最终到账', 'Final credit') }}</span>
      <span class="text-lg font-bold text-primary-600 dark:text-primary-400">${{ promotion.finalCredit.toFixed(2) }}</span>
    </div>
    <p class="text-xs text-gray-500 dark:text-gray-400">{{ localText('活动赠送以创建订单时服务端计算结果为准。', 'The server calculation when the order is created is authoritative.') }}</p>
  </template>
  <p v-else-if="promotion.minimumHint" class="border-t border-gray-200 pt-2 text-xs text-gray-500 dark:border-dark-600 dark:text-gray-400">
    {{ promotion.minimumHint }}
  </p>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'

const props = defineProps<{ amount: number; creditedAmount: number }>()
const { locale } = useI18n()
const appStore = useAppStore()

function localText(zh: string, en: string): string { return locale.value.startsWith('zh') ? zh : en }
function asNumber(value: unknown): number { return typeof value === 'number' && Number.isFinite(value) ? value : 0 }
function asString(value: unknown): string { return typeof value === 'string' ? value : '' }
function round2(value: number): number { return Math.round((value + Number.EPSILON) * 100) / 100 }

const promotion = computed(() => {
  const settings = appStore.cachedPublicSettings
  const enabled = settings?.recharge_promotion_enabled === true
  const rate = asNumber(settings?.recharge_promotion_bonus_rate)
  const minimum = asNumber(settings?.recharge_promotion_min_amount)
  const maximum = asNumber(settings?.recharge_promotion_max_bonus)
  const start = Date.parse(asString(settings?.recharge_promotion_start_at))
  const end = Date.parse(asString(settings?.recharge_promotion_end_at))
  const now = Date.now()
  const inWindow = (Number.isNaN(start) || now >= start) && (Number.isNaN(end) || now < end)
  const meetsMinimum = props.amount >= minimum
  if (!enabled || rate <= 0 || props.amount <= 0 || props.creditedAmount <= 0 || !inWindow || !meetsMinimum) {
    return {
      active: false,
      bonus: 0,
      finalCredit: props.creditedAmount,
      minimumHint: enabled && minimum > props.amount
        ? localText(`充值满 $${minimum.toFixed(2)} 可享活动赠送。`, `Recharge $${minimum.toFixed(2)} to receive the promotion.`)
        : '',
    }
  }
  const rawBonus = round2(props.creditedAmount * rate / 100)
  const bonus = maximum > 0 ? Math.min(rawBonus, maximum) : rawBonus
  return { active: true, bonus: round2(bonus), finalCredit: round2(props.creditedAmount + bonus), minimumHint: '' }
})
</script>
