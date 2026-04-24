<template>
  <div class="max-w-3xl mx-auto space-y-6 p-4">
    <h2 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('leaderboard.title', '转账排行榜') }}</h2>
    <div class="flex gap-2 mb-4">
      <button v-for="p in periods" :key="p.value" @click="period = p.value"
        :class="['px-3 py-1 rounded text-sm', period === p.value ? 'bg-blue-500 text-white' : 'bg-gray-100 dark:bg-gray-700']">
        {{ p.label }}
      </button>
    </div>
    <div v-if="entries.length === 0" class="text-sm text-gray-500">暂无数据</div>
    <div v-for="(entry, i) in entries" :key="entry.user_id"
      class="flex items-center gap-4 rounded-lg bg-white dark:bg-gray-800 p-4 shadow mb-2">
      <div class="text-2xl font-bold w-8 text-center" :class="i < 3 ? 'text-yellow-500' : 'text-gray-400'">{{ entry.rank }}</div>
      <div class="flex-1">
        <div class="text-sm font-medium">{{ entry.email }}</div>
        <div class="text-xs text-gray-500">{{ entry.total_count }} 笔</div>
      </div>
      <div class="text-right">
        <div class="font-bold text-blue-600">{{ entry.total_amount.toFixed(4) }}</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { getTransferLeaderboard } from '@/api/transfer'
import type { TransferLeaderboardEntry } from '@/api/transfer'

const { t } = useI18n()
const period = ref('day')
const entries = ref<TransferLeaderboardEntry[]>([])

const periods = [
  { label: '日榜', value: 'day' },
  { label: '周榜', value: 'week' },
  { label: '月榜', value: 'month' },
]

async function loadLeaderboard() {
  try {
    entries.value = await getTransferLeaderboard({ period: period.value, limit: 20 })
  } catch {}
}

watch(period, loadLeaderboard)
onMounted(loadLeaderboard)
</script>
