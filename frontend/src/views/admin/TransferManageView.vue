<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h2 class="text-xl font-bold">{{ t('nav.transferManage', '转账管理') }}</h2>
      <div class="flex gap-2">
        <button class="btn-primary text-sm" @click="showBatch = true">批量发放</button>
      </div>
    </div>

    <div class="grid grid-cols-3 gap-4" v-if="feeStats.length">
      <div class="rounded-lg bg-green-50 dark:bg-green-900/20 p-4 text-center">
        <div class="text-xs text-gray-500">30天手续费收入</div>
        <div class="text-lg font-bold text-green-600">{{ totalFee.toFixed(4) }}</div>
      </div>
      <div class="rounded-lg bg-blue-50 dark:bg-blue-900/20 p-4 text-center">
        <div class="text-xs text-gray-500">30天总笔数</div>
        <div class="text-lg font-bold text-blue-600">{{ totalCount }}</div>
      </div>
      <div class="rounded-lg bg-purple-50 dark:bg-purple-900/20 p-4 text-center">
        <div class="text-xs text-gray-500">记录总数</div>
        <div class="text-lg font-bold text-purple-600">{{ pagination.total }}</div>
      </div>
    </div>

    <div class="overflow-x-auto rounded-lg bg-white dark:bg-gray-800 shadow">
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
          <tr v-for="t in transfers" :key="t.id" class="border-t dark:border-gray-700">
            <td class="px-4 py-3">{{ t.id }}</td>
            <td class="px-4 py-3">{{ t.sender_id }}</td>
            <td class="px-4 py-3">{{ t.receiver_id }}</td>
            <td class="px-4 py-3 text-right">{{ t.amount.toFixed(4) }}</td>
            <td class="px-4 py-3 text-right">{{ t.fee.toFixed(4) }}</td>
            <td class="px-4 py-3">{{ t.transfer_type }}</td>
            <td class="px-4 py-3">
              <span :class="statusClass(t.status)">{{ t.status }}</span>
            </td>
            <td class="px-4 py-3 text-xs text-gray-500">{{ new Date(t.created_at).toLocaleString() }}</td>
            <td class="px-4 py-3">
              <template v-if="t.status === 'completed'">
                <button class="text-yellow-600 text-xs mr-2" @click="handleFreeze(t.id)">冻结</button>
                <button class="text-red-600 text-xs" @click="handleRevoke(t.id)">撤回</button>
              </template>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showBatch" class="fixed inset-0 bg-black/50 flex items-center justify-center z-50" @click.self="showBatch = false">
      <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-[500px] max-h-[80vh] overflow-auto">
        <h3 class="text-lg font-semibold mb-4">批量发放余额</h3>
        <div class="space-y-3">
          <div v-for="(target, i) in batchTargets" :key="i" class="flex gap-2">
            <input v-model.number="target.user_id" type="number" placeholder="用户ID" class="input-field flex-1" />
            <input v-model.number="target.amount" type="number" step="0.01" placeholder="金额" class="input-field flex-1" />
            <button class="text-red-500" @click="batchTargets.splice(i, 1)">✕</button>
          </div>
        </div>
        <button class="text-blue-500 text-sm mt-2" @click="batchTargets.push({ user_id: 0, amount: 0 })">+ 添加</button>
        <input v-model="batchMemo" type="text" placeholder="备注(可选)" class="input-field w-full mt-3" />
        <div class="flex gap-2 mt-4">
          <button class="btn-primary flex-1" @click="handleBatch" :disabled="batchLoading">{{ batchLoading ? '发放中...' : '确认发放' }}</button>
          <button class="btn-secondary flex-1" @click="showBatch = false">取消</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { TransferRecord, DailyFeeStat } from '@/api/admin/transfer'

const { t } = useI18n()
const transfers = ref<TransferRecord[]>([])
const feeStats = ref<DailyFeeStat[]>([])
const showBatch = ref(false)
const batchTargets = reactive([{ user_id: 0, amount: 0 }])
const batchMemo = ref('')
const batchLoading = ref(false)
const pagination = reactive({ total: 0, page: 1, page_size: 20 })

const totalFee = computed(() => feeStats.value.reduce((s, d) => s + d.total_fee, 0))
const totalCount = computed(() => feeStats.value.reduce((s, d) => s + d.count, 0))

onMounted(async () => {
  await Promise.all([loadTransfers(), loadFeeStats()])
})

async function loadTransfers() {
  try {
    const res = await adminAPI.transfer.listTransfers({ page: pagination.page, page_size: pagination.page_size })
    transfers.value = res.items || []
    pagination.total = res.total
  } catch {}
}

async function loadFeeStats() {
  try {
    feeStats.value = await adminAPI.transfer.getFeeStats({})
  } catch {}
}

function statusClass(status: string) {
  switch (status) {
    case 'completed': return 'text-green-600'
    case 'frozen': return 'text-yellow-600'
    case 'revoked': return 'text-red-600'
    default: return 'text-gray-500'
  }
}

async function handleFreeze(id: number) {
  if (!confirm('确认冻结此转账？')) return
  try {
    await adminAPI.transfer.freezeTransfer(id)
    loadTransfers()
  } catch {}
}

async function handleRevoke(id: number) {
  const reason = prompt('请输入撤回原因:')
  if (!reason) return
  try {
    await adminAPI.transfer.revokeTransfer(id, reason)
    loadTransfers()
  } catch {}
}

async function handleBatch() {
  batchLoading.value = true
  try {
    const valid = batchTargets.filter(t => t.user_id > 0 && t.amount > 0)
    await adminAPI.transfer.batchDistribute(valid, batchMemo.value || undefined)
    showBatch.value = false
    loadTransfers()
  } catch {} finally {
    batchLoading.value = false
  }
}
</script>
