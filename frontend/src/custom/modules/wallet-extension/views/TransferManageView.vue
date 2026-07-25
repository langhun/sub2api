<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <h2 class="text-xl font-bold">{{ t('nav.transferManage', '转账管理') }}</h2>
        <div class="flex gap-2">
          <button class="btn-primary text-sm" @click="showBatch = true">批量发放</button>
        </div>
      </div>

      <div v-if="feeStats.length" class="grid grid-cols-3 gap-4">
        <div class="rounded-lg bg-green-50 p-4 text-center dark:bg-green-900/20">
          <div class="text-xs text-gray-500">30天手续费收入</div>
          <div class="text-lg font-bold text-green-600">{{ totalFee.toFixed(4) }}</div>
        </div>
        <div class="rounded-lg bg-blue-50 p-4 text-center dark:bg-blue-900/20">
          <div class="text-xs text-gray-500">30天总笔数</div>
          <div class="text-lg font-bold text-blue-600">{{ totalCount }}</div>
        </div>
        <div class="rounded-lg bg-purple-50 p-4 text-center dark:bg-purple-900/20">
          <div class="text-xs text-gray-500">记录总数</div>
          <div class="text-lg font-bold text-purple-600">{{ pagination.total }}</div>
        </div>
      </div>

      <div class="overflow-x-auto rounded-lg bg-white shadow dark:bg-gray-800">
        <table class="w-full text-sm">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-4 py-3 text-left">ID</th>
              <th class="px-4 py-3 text-left">发送方</th>
              <th class="px-4 py-3 text-left">接收方</th>
              <th class="px-4 py-3 text-right">金额</th>
              <th class="px-4 py-3 text-right">手续费</th>
              <th class="px-4 py-3 text-left">类型</th>
              <th class="px-4 py-3 text-left">状态</th>
              <th class="px-4 py-3 text-left">时间</th>
              <th class="px-4 py-3 text-left">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="transfer in transfers" :key="transfer.id" class="border-t dark:border-gray-700">
              <td class="px-4 py-3">{{ transfer.id }}</td>
              <td class="px-4 py-3">{{ transfer.sender_display }}</td>
              <td class="px-4 py-3">{{ transfer.receiver_display }}</td>
              <td class="px-4 py-3 text-right">{{ transfer.amount.toFixed(4) }}</td>
              <td class="px-4 py-3 text-right">{{ transfer.fee.toFixed(4) }}</td>
              <td class="px-4 py-3">{{ transferTypeLabel(transfer.transfer_type) }}</td>
              <td class="px-4 py-3">
                <span :class="statusClass(transfer.status)">{{ transferStatusLabel(transfer.status) }}</span>
              </td>
              <td class="px-4 py-3 text-xs text-gray-500">{{ new Date(transfer.created_at).toLocaleString() }}</td>
              <td class="px-4 py-3">
                <template v-if="transfer.status === 'completed'">
                  <button class="mr-2 text-xs text-yellow-600" @click="handleFreeze(transfer.id)">冻结</button>
                  <button class="text-xs text-red-600" @click="handleRevoke(transfer.id)">撤回</button>
                </template>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="showBatch" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50" @click.self="showBatch = false">
        <div class="max-h-[80vh] w-[500px] overflow-auto rounded-lg bg-white p-6 dark:bg-gray-800">
          <h3 class="mb-4 text-lg font-semibold">批量发放余额</h3>
          <div class="space-y-3">
            <div v-for="(target, index) in batchTargets" :key="index" class="flex gap-2">
              <input v-model.number="target.user_id" type="number" placeholder="用户ID" class="input-field flex-1" />
              <input v-model.number="target.amount" type="number" step="0.01" placeholder="金额" class="input-field flex-1" />
              <button class="text-red-500" @click="batchTargets.splice(index, 1)">✕</button>
            </div>
          </div>
          <button class="mt-2 text-sm text-blue-500" @click="batchTargets.push({ user_id: 0, amount: 0 })">+ 添加</button>
          <input v-model="batchMemo" type="text" placeholder="备注(可选)" class="input-field mt-3 w-full" />
          <div class="mt-4 flex gap-2">
            <button class="btn-primary flex-1" :disabled="batchLoading" @click="handleBatch">{{ batchLoading ? '发放中...' : '确认发放' }}</button>
            <button class="btn-secondary flex-1" @click="showBatch = false">取消</button>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import type { DirectTransferRecord } from '../api'
import { adminTransferAPI, type DailyFeeStat } from '../api/admin'

const { t } = useI18n()
const transfers = ref<DirectTransferRecord[]>([])
const feeStats = ref<DailyFeeStat[]>([])
const showBatch = ref(false)
const batchTargets = reactive([{ user_id: 0, amount: 0 }])
const batchMemo = ref('')
const batchLoading = ref(false)
const pagination = reactive({ total: 0, page: 1, page_size: 20 })

const totalFee = computed(() => feeStats.value.reduce((sum, stat) => sum + stat.total_fee, 0))
const totalCount = computed(() => feeStats.value.reduce((sum, stat) => sum + stat.count, 0))

onMounted(async () => {
  await Promise.all([loadTransfers(), loadFeeStats()])
})

async function loadTransfers() {
  try {
    const result = await adminTransferAPI.listTransfers({ page: pagination.page, page_size: pagination.page_size })
    transfers.value = result.items || []
    pagination.total = result.total
  } catch (error) {
    console.error('Failed to load transfers', error)
  }
}

async function loadFeeStats() {
  try {
    feeStats.value = await adminTransferAPI.getFeeStats({})
  } catch (error) {
    console.error('Failed to load transfer fee statistics', error)
  }
}

function statusClass(status: string) {
  switch (status) {
    case 'completed': return 'text-green-600'
    case 'frozen': return 'text-yellow-600'
    case 'revoked': return 'text-red-600'
    default: return 'text-gray-500'
  }
}

function transferTypeLabel(type: string) {
  return t(`transfer.transferTypes.${type}`, type)
}

function transferStatusLabel(status: string) {
  return t(`transfer.status.${status}`, status)
}

async function handleFreeze(id: number) {
  if (!confirm('确认冻结此转账？')) return
  try {
    await adminTransferAPI.freezeTransfer(id)
    loadTransfers()
  } catch (error) {
    console.error('Failed to freeze transfer', error)
  }
}

async function handleRevoke(id: number) {
  const reason = prompt('请输入撤回原因:')
  if (!reason) return
  try {
    await adminTransferAPI.revokeTransfer(id, reason)
    loadTransfers()
  } catch (error) {
    console.error('Failed to revoke transfer', error)
  }
}

async function handleBatch() {
  batchLoading.value = true
  try {
    const validTargets = batchTargets.filter((target) => target.user_id > 0 && target.amount > 0)
    await adminTransferAPI.batchDistribute(validTargets, batchMemo.value || undefined)
    showBatch.value = false
    loadTransfers()
  } catch (error) {
    console.error('Failed to distribute balances', error)
  } finally {
    batchLoading.value = false
  }
}
</script>
