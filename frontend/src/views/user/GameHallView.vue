<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <header class="space-y-2">
        <p class="text-sm font-medium text-gray-500 dark:text-gray-400">DG Wallet</p>
        <h1 class="text-3xl font-bold text-gray-900 dark:text-white">娱乐大厅</h1>
        <p class="text-sm text-gray-500 dark:text-gray-400">主余额与 DG 余额独立管理，支持 1:1 兑换。</p>
      </header>

      <p
        v-if="errorText"
        class="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-300"
      >
        {{ errorText }}
      </p>

      <section class="grid gap-4 md:grid-cols-3">
        <article class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <p class="text-sm text-gray-500 dark:text-gray-400">主余额</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatMainBalance(hallStatus?.main_balance) }}</p>
        </article>
        <article class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <p class="text-sm text-gray-500 dark:text-gray-400">DG 余额</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatDG(hallStatus?.dg_balance) }}</p>
        </article>
        <article class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <p class="text-sm text-gray-500 dark:text-gray-400">大厅奖池</p>
          <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ formatDG(hallStatus?.jackpot_balance) }}</p>
        </article>
      </section>

      <section class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-4 md:flex-row md:items-end">
          <label class="flex-1">
            <span class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">兑换方向</span>
            <select v-model="exchangeDirection" class="input">
              <option value="balance_to_dg">主余额转 DG</option>
              <option value="dg_to_balance">DG 转主余额</option>
            </select>
          </label>
          <label class="flex-1">
            <span class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">兑换数量</span>
            <input v-model.number="exchangeAmount" type="number" min="0.01" step="0.01" class="input" />
          </label>
          <button class="btn bg-primary-600 text-white hover:bg-primary-700" :disabled="exchangeSubmitting" @click="submitExchange">
            {{ exchangeSubmitting ? '兑换中...' : '立即兑换' }}
          </button>
        </div>
      </section>

      <section class="grid gap-4 md:grid-cols-2">
        <RouterLink
          to="/games/slots"
          data-testid="slots-entry"
          class="rounded-xl border border-gray-200 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary-300 hover:shadow-md dark:border-dark-700 dark:bg-dark-900"
        >
          <div class="flex items-center justify-between gap-4">
            <div>
              <h2 class="text-xl font-semibold text-gray-900 dark:text-white">老虎机</h2>
              <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">使用 DG 余额进行即时投注与结算。</p>
            </div>
            <span class="rounded-full bg-amber-100 px-3 py-1 text-sm font-semibold text-amber-700 dark:bg-amber-950/40 dark:text-amber-300">7</span>
          </div>
        </RouterLink>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { gamesAPI, type GameExchangeDirection, type GameHallStatus } from '@/api/games'
import { useAppStore } from '@/stores'
import { formatNumber } from '@/utils/format'

const appStore = useAppStore()

const hallStatus = ref<GameHallStatus | null>(null)
const errorText = ref('')
const exchangeSubmitting = ref(false)
const exchangeDirection = ref<GameExchangeDirection>('balance_to_dg')
const exchangeAmount = ref(10)

onMounted(() => {
  void loadHall()
})

async function loadHall() {
  errorText.value = ''
  try {
    hallStatus.value = await gamesAPI.getHall()
  } catch (error) {
    errorText.value = toErrorMessage(error)
    appStore.showError(errorText.value)
  }
}

async function submitExchange() {
  if (exchangeSubmitting.value || exchangeAmount.value <= 0) return

  exchangeSubmitting.value = true
  errorText.value = ''
  try {
    const result = await gamesAPI.exchange(exchangeDirection.value, exchangeAmount.value)
    if (!hallStatus.value) {
      hallStatus.value = {
        main_balance: result.main_balance_after,
        dg_balance: result.dg_balance_after,
        jackpot_balance: 0,
        games: [],
      }
    } else {
      hallStatus.value.main_balance = result.main_balance_after
      hallStatus.value.dg_balance = result.dg_balance_after
    }
  } catch (error) {
    errorText.value = toErrorMessage(error)
    appStore.showError(errorText.value)
  } finally {
    exchangeSubmitting.value = false
  }
}

function formatDG(value: number | null | undefined) {
  const amount = Number(value ?? 0)
  return `${formatNumber(Number.isFinite(amount) ? amount : 0)} DG`
}

function formatMainBalance(value: number | null | undefined) {
  const amount = Number(value ?? 0)
  return formatNumber(Number.isFinite(amount) ? amount : 0)
}

function toErrorMessage(error: unknown) {
  return (error as { message?: string })?.message || '娱乐大厅加载失败，请稍后再试。'
}
</script>
