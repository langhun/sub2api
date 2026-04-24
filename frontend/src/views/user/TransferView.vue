<template>
  <div class="max-w-2xl mx-auto space-y-6 p-4">
    <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('transfer.title', '余额转账') }}</h2>

    <div v-if="stats" class="grid grid-cols-3 gap-4">
      <div class="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-4 text-center">
        <div class="text-xs text-gray-500">{{ t('transfer.totalSent', '累计转出') }}</div>
        <div class="text-lg font-bold text-blue-600">{{ stats.total_sent.toFixed(4) }}</div>
      </div>
      <div class="rounded-lg bg-green-50 dark:bg-green-900/20 p-4 text-center">
        <div class="text-xs text-gray-500">{{ t('transfer.totalReceived', '累计转入') }}</div>
        <div class="text-lg font-bold text-green-600">{{ stats.total_received.toFixed(4) }}</div>
      </div>
      <div class="rounded-lg bg-orange-50 dark:bg-orange-900/20 p-4 text-center">
        <div class="text-xs text-gray-500">{{ t('transfer.totalFee', '手续费') }}</div>
        <div class="text-lg font-bold text-orange-600">{{ stats.total_fee_paid.toFixed(4) }}</div>
      </div>
    </div>

    <form @submit.prevent="handleTransfer" class="space-y-4 rounded-lg bg-white dark:bg-gray-800 p-6 shadow">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ t('transfer.receiverId', '接收方用户 ID') }}
        </label>
        <input v-model.number="form.receiverId" type="number" required min="1" class="input-field w-full" />
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ t('transfer.amount', '金额') }}
        </label>
        <input v-model.number="form.amount" type="number" step="0.01" required min="0.01" class="input-field w-full" @input="calcFee" />
      </div>
      <div v-if="feePreview !== null" class="text-sm text-gray-500">
        {{ t('transfer.feePreview', '手续费') }}: {{ feePreview.toFixed(4) }} | {{ t('transfer.total', '合计扣款') }}: {{ (form.amount + feePreview).toFixed(4) }}
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ t('transfer.memo', '留言') }}
        </label>
        <input v-model="form.memo" type="text" maxlength="200" class="input-field w-full" />
      </div>
      <div v-if="error" class="text-sm text-red-500">{{ error }}</div>
      <div v-if="success" class="text-sm text-green-500">{{ t('transfer.success', '转账成功') }}</div>
      <button type="submit" :disabled="loading" class="btn-primary w-full">
        {{ loading ? t('common.saving') : t('transfer.submit', '确认转账') }}
      </button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { transferBalance, getTransferStats, validateTransfer } from '@/api'
import type { TransferStats } from '@/api/transfer'

const { t } = useI18n()
const stats = ref<TransferStats | null>(null)
const feePreview = ref<number | null>(null)
const loading = ref(false)
const error = ref('')
const success = ref(false)

const form = reactive({
  receiverId: 0,
  amount: 0,
  memo: '',
})

onMounted(async () => {
  try {
    stats.value = await getTransferStats()
  } catch {}
})

async function calcFee() {
  if (form.receiverId > 0 && form.amount > 0) {
    try {
      const result = await validateTransfer(form.receiverId, form.amount)
      feePreview.value = result.fee
    } catch {
      feePreview.value = null
    }
  }
}

async function handleTransfer() {
  error.value = ''
  success.value = false
  loading.value = true
  try {
    await transferBalance(form.receiverId, form.amount, form.memo || undefined)
    success.value = true
    form.receiverId = 0
    form.amount = 0
    form.memo = ''
    feePreview.value = null
    stats.value = await getTransferStats()
  } catch (e: any) {
    error.value = e?.response?.data?.error || t('transfer.failed', '转账失败')
  } finally {
    loading.value = false
  }
}
</script>
