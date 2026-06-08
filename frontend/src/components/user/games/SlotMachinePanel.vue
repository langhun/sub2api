<template>
  <section class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_22rem]">
    <div class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <div class="flex flex-col gap-4 border-b border-gray-100 pb-5 dark:border-dark-700 md:flex-row md:items-end md:justify-between">
        <div>
          <p class="text-sm font-medium text-gray-500 dark:text-gray-400">老虎机</p>
          <h2 class="text-3xl font-bold text-gray-900 dark:text-white">DG 老虎机</h2>
        </div>
        <div class="text-sm text-gray-500 dark:text-gray-400">单次投注，立即结算</div>
      </div>

      <div class="mt-5 grid gap-4 md:grid-cols-2">
        <article class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
          <p class="text-sm text-gray-500 dark:text-gray-400">DG 余额</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatDG(currentDGBalance) }}</p>
        </article>
        <article class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800">
          <p class="text-sm text-gray-500 dark:text-gray-400">大厅奖池</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatDG(currentJackpot) }}</p>
        </article>
      </div>

      <div class="mt-6 grid gap-4 md:grid-cols-[minmax(0,1fr)_auto]">
        <label>
          <span class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">投注金额</span>
          <input
            v-model.number="betAmount"
            data-testid="slot-bet-input"
            type="number"
            min="0.01"
            step="0.01"
            class="input"
          />
        </label>
        <button
          data-testid="slot-spin"
          class="btn self-end bg-primary-600 text-white hover:bg-primary-700"
          :disabled="submitting"
          @click="playSlots"
        >
          {{ submitting ? '旋转中...' : '开始旋转' }}
        </button>
      </div>

      <div class="mt-6 rounded-xl border border-dashed border-gray-300 bg-gray-50 p-5 dark:border-dark-600 dark:bg-dark-800">
        <p class="text-sm text-gray-500 dark:text-gray-400">最近图案</p>
        <div class="mt-3 flex gap-3 text-3xl">
          <span v-for="(symbol, index) in currentSymbols" :key="`${symbol}-${index}`">{{ symbolEmoji(symbol) }}</span>
        </div>
      </div>

      <div
        data-testid="slot-result-message"
        class="mt-6 rounded-xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-700 dark:border-dark-700 dark:bg-dark-900 dark:text-gray-200"
      >
        {{ resultMessage }}
      </div>
    </div>

    <aside class="rounded-xl border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">历史记录</h3>
      <div class="mt-4 space-y-3">
        <div v-if="history.length === 0" class="text-sm text-gray-500 dark:text-gray-400">暂无记录</div>
        <div v-for="item in history" :key="item.id" class="rounded-lg border border-gray-200 p-3 dark:border-dark-700">
          <div class="flex items-center justify-between gap-3">
            <span class="text-sm text-gray-500 dark:text-gray-400">{{ item.time }}</span>
            <span class="text-sm font-semibold text-gray-900 dark:text-white">{{ formatDG(item.dgAfter) }}</span>
          </div>
          <div class="mt-2 flex gap-2 text-2xl">
            <span v-for="(symbol, index) in item.symbols" :key="`${item.id}-${index}`">{{ symbolEmoji(symbol) }}</span>
          </div>
          <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ item.message }}</p>
        </div>
      </div>
    </aside>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { gamesAPI, type GameHallStatus, type GamePlayResult } from '@/api/games'
import { useAppStore } from '@/stores'
import { formatNumber } from '@/utils/format'

interface SlotHistoryItem {
  id: number
  dgAfter: number
  message: string
  symbols: string[]
  time: string
}

const appStore = useAppStore()

const hallStatus = ref<GameHallStatus | null>(null)
const submitting = ref(false)
const betAmount = ref(10)
const resultMessage = ref('等待转动老虎机')
const currentSymbols = ref<string[]>(['cherry', 'lemon', 'bell'])
const history = ref<SlotHistoryItem[]>([])
let historyID = 0

const currentDGBalance = ref(0)
const currentJackpot = ref(0)

onMounted(() => {
  void loadHall()
})

async function loadHall() {
  try {
    hallStatus.value = await gamesAPI.getHall()
    currentDGBalance.value = hallStatus.value.dg_balance
    currentJackpot.value = hallStatus.value.jackpot_balance
  } catch (error) {
    resultMessage.value = toErrorMessage(error)
    appStore.showError(resultMessage.value)
  }
}

async function playSlots() {
  if (submitting.value || betAmount.value <= 0) return

  submitting.value = true
  try {
    const result = await gamesAPI.play('slots', Number(betAmount.value))
    applyResult(result)
  } catch (error) {
    resultMessage.value = toErrorMessage(error)
    appStore.showError(resultMessage.value)
  } finally {
    submitting.value = false
  }
}

function applyResult(result: GamePlayResult) {
  currentDGBalance.value = result.dg_balance_after
  currentJackpot.value = result.jackpot_balance
  currentSymbols.value = result.symbols?.length ? result.symbols : ['cherry', 'lemon', 'bell']
  resultMessage.value = result.message
  history.value.unshift({
    id: ++historyID,
    dgAfter: result.dg_balance_after,
    message: result.message,
    symbols: [...currentSymbols.value],
    time: new Date().toLocaleTimeString('zh-CN', { hour12: false }),
  })
  history.value = history.value.slice(0, 8)
}

function formatDG(value: number | null | undefined) {
  const amount = Number(value ?? 0)
  return `${formatNumber(Number.isFinite(amount) ? amount : 0)} DG`
}

function symbolEmoji(symbol: string) {
  switch (symbol) {
    case 'cherry':
      return '🍒'
    case 'lemon':
      return '🍋'
    case 'orange':
      return '🍊'
    case 'bell':
      return '🔔'
    case 'grape':
      return '🍇'
    case 'star':
      return '⭐'
    case 'diamond':
      return '💎'
    case '7':
      return '7️⃣'
    default:
      return '🎰'
  }
}

function toErrorMessage(error: unknown) {
  return (error as { message?: string })?.message || '老虎机暂时不可用，请稍后重试。'
}
</script>
