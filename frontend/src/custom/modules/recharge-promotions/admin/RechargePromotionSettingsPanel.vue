<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ localText('充值赠送活动', 'Recharge promotion') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ localText('支付金额不变，按基础到账余额额外赠送。活动关闭不会影响已有订单。', 'Adds balance based on credited amount without changing the payment amount. Existing orders are unchanged when disabled.') }}</p>
    </div>
    <div class="space-y-5 p-6">
      <div class="flex items-center justify-between gap-4">
        <div>
          <p class="font-medium text-gray-900 dark:text-white">{{ localText('启用充值赠送', 'Enable recharge promotion') }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ localText('仅影响新创建的余额充值订单。', 'Only affects newly created balance orders.') }}</p>
        </div>
        <Toggle v-model="form.recharge_promotion_enabled" />
      </div>
      <div class="grid gap-4 border-t border-gray-100 pt-4 dark:border-dark-700 md:grid-cols-3">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('赠送比例（%）', 'Bonus rate (%)') }}<input v-model.number="form.recharge_promotion_bonus_rate" type="number" min="0" max="100" step="0.01" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('最低充值金额', 'Minimum recharge') }}<input v-model.number="form.recharge_promotion_min_amount" type="number" min="0" step="0.01" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('单笔最高赠送', 'Maximum bonus per order') }}<input v-model.number="form.recharge_promotion_max_bonus" type="number" min="0" step="0.01" class="input mt-1" /><span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ localText('0 表示不封顶', '0 means unlimited') }}</span></label>
      </div>
      <div class="grid gap-4 border-t border-gray-100 pt-4 dark:border-dark-700 md:grid-cols-2">
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('开始时间（RFC3339，可空）', 'Start time (RFC3339, optional)') }}<input v-model="form.recharge_promotion_start_at" type="text" placeholder="2026-08-05T00:00:00Z" class="input mt-1" /></label>
        <label class="text-sm text-gray-600 dark:text-gray-300">{{ localText('结束时间（RFC3339，可空）', 'End time (RFC3339, optional)') }}<input v-model="form.recharge_promotion_end_at" type="text" placeholder="2026-08-12T00:00:00Z" class="input mt-1" /></label>
      </div>
      <div class="rounded border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
        {{ localText('赠送不参与邀请返利；最终到账以服务端创建订单时的活动快照为准。', 'Bonus does not participate in affiliate rebates. The server-side order snapshot is authoritative.') }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import type { RechargePromotionSettings } from '../settings'

const props = defineProps<{ modelValue: RechargePromotionSettings }>()
const emit = defineEmits<{ 'update:modelValue': [value: RechargePromotionSettings] }>()
const { locale } = useI18n()
const form = reactive<RechargePromotionSettings>({ ...props.modelValue })

function localText(zh: string, en: string): string { return locale.value.startsWith('zh') ? zh : en }

watch(() => props.modelValue, (value) => Object.assign(form, value), { deep: true })
watch(form, (value) => emit('update:modelValue', { ...value }), { deep: true })
</script>
