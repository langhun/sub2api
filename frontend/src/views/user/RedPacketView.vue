<template>
  <div class="max-w-2xl mx-auto space-y-6 p-4">
    <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('redpacket.title', '红包中心') }}</h2>

    <div class="grid grid-cols-2 gap-4">
      <button class="rounded-lg bg-red-500 text-white py-3 font-medium hover:bg-red-600" @click="showCreate = true">
        {{ t('redpacket.create', '发红包') }}
      </button>
      <button class="rounded-lg bg-yellow-500 text-white py-3 font-medium hover:bg-yellow-600" @click="showClaim = true">
        {{ t('redpacket.claim', '领红包') }}
      </button>
    </div>

    <div v-if="showCreate" class="rounded-lg bg-white dark:bg-gray-800 p-6 shadow space-y-4">
      <h3 class="text-lg font-semibold">{{ t('redpacket.create', '发红包') }}</h3>
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">总金额</label>
        <input v-model.number="createForm.total_amount" type="number" step="0.01" min="0.01" class="input-field w-full" />
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">份数</label>
        <input v-model.number="createForm.count" type="number" min="1" max="100" class="input-field w-full" />
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">类型</label>
        <select v-model="createForm.redpacket_type" class="input-field w-full">
          <option value="equal">等分红包</option>
          <option value="random">拼手气红包</option>
        </select>
      </div>
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">附言</label>
        <input v-model="createForm.memo" type="text" maxlength="100" class="input-field w-full" />
      </div>
      <div v-if="createError" class="text-sm text-red-500">{{ createError }}</div>
      <div class="flex gap-2">
        <button class="btn-primary flex-1" @click="handleCreate" :disabled="createLoading">{{ t('common.saving') }}</button>
        <button class="btn-secondary flex-1" @click="showCreate = false">{{ t('common.cancel', '取消') }}</button>
      </div>
      <div v-if="createdRp" class="rounded bg-green-50 dark:bg-green-900/20 p-4">
        <p class="text-sm font-medium text-green-700">红包已创建！</p>
        <p class="mt-1 font-mono text-lg select-all">{{ createdRp.code }}</p>
        <p class="text-xs text-gray-500 mt-1">将此口令分享给好友即可领取</p>
      </div>
    </div>

    <div v-if="showClaim" class="rounded-lg bg-white dark:bg-gray-800 p-6 shadow space-y-4">
      <h3 class="text-lg font-semibold">{{ t('redpacket.claim', '领红包') }}</h3>
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">红包口令</label>
        <input v-model="claimCode" type="text" class="input-field w-full" placeholder="输入红包口令" />
      </div>
      <div v-if="claimResult" class="rounded bg-green-50 dark:bg-green-900/20 p-4">
        <p class="text-green-700 font-bold text-lg">+{{ claimResult.amount.toFixed(4) }}</p>
      </div>
      <div v-if="claimError" class="text-sm text-red-500">{{ claimError }}</div>
      <div class="flex gap-2">
        <button class="btn-primary flex-1" @click="handleClaim" :disabled="claimLoading">领取</button>
        <button class="btn-secondary flex-1" @click="showClaim = false; claimResult = null; claimError = ''">{{ t('common.cancel', '取消') }}</button>
      </div>
    </div>

    <div>
      <h3 class="text-lg font-semibold mb-3">{{ t('redpacket.myPackets', '我的红包') }}</h3>
      <div v-if="myPackets.length === 0" class="text-sm text-gray-500">暂无红包记录</div>
      <div v-for="rp in myPackets" :key="rp.id" class="rounded-lg bg-white dark:bg-gray-800 p-4 shadow mb-2">
        <div class="flex justify-between items-center">
          <div>
            <span class="text-sm font-medium">{{ rp.redpacket_type === 'equal' ? '等分' : '拼手气' }}红包</span>
            <span class="text-xs text-gray-500 ml-2">{{ rp.total_count }}份</span>
          </div>
          <div class="text-right">
            <div class="font-bold text-red-500">{{ rp.total_amount.toFixed(4) }}</div>
            <div class="text-xs text-gray-500">{{ rp.status }}</div>
          </div>
        </div>
        <div v-if="rp.memo" class="text-xs text-gray-400 mt-1">{{ rp.memo }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { createRedPacket, claimRedPacket, getMyRedPackets } from '@/api/transfer'
import type { RedPacketRecord, RedPacketClaimRecord } from '@/api/transfer'

const { t } = useI18n()
const showCreate = ref(false)
const showClaim = ref(false)
const claimCode = ref('')
const claimResult = ref<RedPacketClaimRecord | null>(null)
const claimError = ref('')
const claimLoading = ref(false)
const createError = ref('')
const createLoading = ref(false)
const createdRp = ref<RedPacketRecord | null>(null)
const myPackets = ref<RedPacketRecord[]>([])

const createForm = reactive({
  total_amount: 0,
  count: 1,
  redpacket_type: 'equal' as 'equal' | 'random',
  memo: '',
})

onMounted(loadMyPackets)

async function loadMyPackets() {
  try {
    const res = await getMyRedPackets({ page: 1, page_size: 20 })
    myPackets.value = res.items || []
  } catch {}
}

async function handleCreate() {
  createError.value = ''
  createLoading.value = true
  try {
    createdRp.value = await createRedPacket({
      total_amount: createForm.total_amount,
      count: createForm.count,
      redpacket_type: createForm.redpacket_type,
      memo: createForm.memo || undefined,
    })
    loadMyPackets()
  } catch (e: any) {
    createError.value = e?.response?.data?.error || '创建失败'
  } finally {
    createLoading.value = false
  }
}

async function handleClaim() {
  claimError.value = ''
  claimResult.value = null
  claimLoading.value = true
  try {
    claimResult.value = await claimRedPacket(claimCode.value)
    loadMyPackets()
  } catch (e: any) {
    claimError.value = e?.response?.data?.error || '领取失败'
  } finally {
    claimLoading.value = false
  }
}
</script>
