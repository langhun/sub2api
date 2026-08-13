<template>
  <div class="space-y-5 border-t border-gray-100 p-6 dark:border-dark-700">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h3 class="font-medium text-gray-900 dark:text-white">{{ t('admin.rechargePromotion.summary') }}</h3>
        <p v-if="summary" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ summary.enabled ? t('admin.rechargePromotion.enabledStatus') : t('admin.rechargePromotion.disabledStatus') }}</p>
      </div>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">{{ t('admin.rechargePromotion.refresh') }}</button>
    </div>

    <div v-if="summary" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      <Metric :label="t('admin.rechargePromotion.promotedOrders')" :value="String(summary.orders)" />
      <Metric :label="t('admin.rechargePromotion.totalBonus')" :value="`$${summary.bonus_credit.toFixed(2)}`" accent="emerald" />
      <Metric :label="t('admin.rechargePromotion.totalFinalCredit')" :value="`$${summary.final_credit.toFixed(2)}`" />
      <Metric :label="t('admin.rechargePromotion.refundedBonus')" :value="`$${summary.refunded_bonus_credit.toFixed(2)}`" accent="red" />
    </div>
    <p v-if="loadFailed" data-testid="recharge-promotion-load-failed" class="text-sm text-red-600 dark:text-red-300">{{ t('admin.rechargePromotion.loadFailed') }}</p>

    <div class="border-t border-gray-100 pt-5 dark:border-dark-700">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
        <h3 class="font-medium text-gray-900 dark:text-white">{{ t('admin.rechargePromotion.details') }}</h3>
        <select v-model="mode" data-testid="recharge-promotion-mode-filter" class="input w-44" @change="handleModeChange">
          <option value="">{{ t('admin.rechargePromotion.allModes') }}</option>
          <option value="percentage">{{ t('admin.rechargePromotion.percentageMode') }}</option>
          <option value="tiered">{{ t('admin.rechargePromotion.tieredMode') }}</option>
          <option value="tiered_percentage">{{ t('admin.rechargePromotion.tieredPercentageMode') }}</option>
        </select>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[680px] text-left text-sm">
          <thead class="border-b border-gray-200 text-xs text-gray-500 dark:border-dark-700 dark:text-gray-400"><tr><th class="py-2 pr-3">{{ t('admin.rechargePromotion.order') }}</th><th class="py-2 pr-3">{{ t('admin.rechargePromotion.user') }}</th><th class="py-2 pr-3">{{ t('admin.rechargePromotion.baseCredit') }}</th><th class="py-2 pr-3">{{ t('admin.rechargePromotion.bonusCredit') }}</th><th class="py-2 pr-3">{{ t('admin.rechargePromotion.finalCredit') }}</th><th class="py-2 pr-3">{{ t('admin.rechargePromotion.refundedCredit') }}</th><th class="py-2">{{ t('admin.rechargePromotion.completedAt') }}</th></tr></thead>
          <tbody>
            <tr v-for="item in items" :key="item.payment_order_id" class="border-b border-gray-100 dark:border-dark-700"><td class="py-2 pr-3 font-mono">#{{ item.payment_order_id }}</td><td class="py-2 pr-3">#{{ item.user_id }}</td><td class="py-2 pr-3 tabular-nums">${{ item.base_credit.toFixed(2) }}</td><td class="py-2 pr-3 tabular-nums text-emerald-700 dark:text-emerald-300">+${{ item.bonus_credit.toFixed(2) }}</td><td class="py-2 pr-3 tabular-nums">${{ item.final_credit.toFixed(2) }}</td><td class="py-2 pr-3 tabular-nums text-red-600 dark:text-red-300">${{ item.refunded_credit.toFixed(2) }}</td><td class="py-2 text-xs text-gray-500">{{ formatDate(item.completed_at) }}</td></tr>
            <tr v-if="!loading && items.length === 0"><td colspan="7" class="py-5 text-center text-gray-500 dark:text-gray-400">-</td></tr>
          </tbody>
        </table>
      </div>
      <div v-if="total > pageSize" class="mt-3 flex justify-end gap-2"><button type="button" class="btn btn-secondary btn-sm" :disabled="page <= 1 || loading" @click="page--; loadOrders()">&lt;</button><span class="px-2 py-1 text-xs text-gray-500">{{ page }}</span><button type="button" class="btn btn-secondary btn-sm" :disabled="page * pageSize >= total || loading" @click="page++; loadOrders()">&gt;</button></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { rechargePromotionsAPI, type RechargePromotionMode, type RechargePromotionOrder, type RechargePromotionSummary } from '../api'

const Metric = defineComponent({
  props: { label: { type: String, required: true }, value: { type: String, required: true }, accent: { type: String, default: '' } },
  setup(props) {
    return () => h('div', { class: 'rounded-lg border border-gray-200 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-800/60' }, [
      h('p', { class: 'text-xs text-gray-500 dark:text-gray-400' }, props.label),
      h('p', { class: ['mt-1 font-semibold tabular-nums', props.accent === 'emerald' ? 'text-emerald-700 dark:text-emerald-300' : props.accent === 'red' ? 'text-red-600 dark:text-red-300' : 'text-gray-900 dark:text-white'] }, props.value),
    ])
  },
})

const { t } = useI18n()
const summary = ref<RechargePromotionSummary | null>(null)
const items = ref<RechargePromotionOrder[]>([])
const loading = ref(false)
const loadFailed = ref(false)
const mode = ref<RechargePromotionMode | ''>('')
const page = ref(1)
const pageSize = 10
const total = ref(0)

function formatDate(value: string) { return new Date(value).toLocaleString() }
async function loadOrders() {
  loading.value = true
  try {
    const response = await rechargePromotionsAPI.getAdminOrders({ page: page.value, page_size: pageSize, mode: mode.value || undefined })
    items.value = response.data.items || []
    total.value = response.data.total || 0
  } catch {
    items.value = []
    total.value = 0
    loadFailed.value = true
  } finally { loading.value = false }
}
function handleModeChange() {
  page.value = 1
  void loadOrders()
}
async function load() {
  loading.value = true
  loadFailed.value = false
  try {
    const response = await rechargePromotionsAPI.getSummary()
    summary.value = response.data
  } catch {
    summary.value = null
    loadFailed.value = true
  } finally {
    loading.value = false
  }
  await loadOrders()
}
onMounted(load)
</script>
