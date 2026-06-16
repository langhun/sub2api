<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <header class="space-y-3">
        <p class="text-sm font-medium uppercase tracking-[0.28em] text-primary-600 dark:text-primary-400">DG Wallet</p>
        <div class="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
          <div class="space-y-2">
            <h1 class="text-3xl font-bold text-gray-900 dark:text-white">娱乐大厅</h1>
            <p class="max-w-2xl text-sm leading-6 text-gray-500 dark:text-gray-400">
              先兑换 DG 币，再进入游戏区体验老虎机等娱乐项目。
            </p>
          </div>
          <div class="inline-flex items-center gap-2 self-start rounded-full border border-primary-200 bg-primary-50 px-4 py-2 text-xs font-medium text-primary-700 dark:border-primary-900/40 dark:bg-primary-900/20 dark:text-primary-300">
            <span class="h-2 w-2 rounded-full bg-primary-500"></span>
            大厅功能已启用
          </div>
        </div>
      </header>

      <p
        v-if="errorText"
        class="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-300"
      >
        {{ errorText }}
      </p>

      <section data-testid="hall-summary" class="grid gap-4 md:grid-cols-2">
        <article class="card border-primary-100/70 bg-gradient-to-br from-primary-50 via-white to-white p-5 dark:border-primary-900/30 dark:from-primary-950/20 dark:via-dark-900 dark:to-dark-900">
          <p class="text-sm text-gray-500 dark:text-gray-400">DG 余额</p>
          <p class="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">{{ formatDG(hallStatus?.dg_balance) }}</p>
          <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">进入游戏前需先持有 DG 币</p>
        </article>
        <article class="card border-primary-100/70 bg-gradient-to-br from-white via-amber-50/60 to-white p-5 dark:border-primary-900/30 dark:from-dark-900 dark:via-amber-950/10 dark:to-dark-900">
          <p class="text-sm text-gray-500 dark:text-gray-400">大厅奖池</p>
          <p class="mt-2 text-3xl font-semibold text-gray-900 dark:text-white">{{ formatDG(hallStatus?.jackpot_balance) }}</p>
          <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">游戏结算时会实时回写奖池余额</p>
        </article>
      </section>

      <section class="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <article class="card overflow-hidden border-primary-100/70 dark:border-primary-900/30">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">游戏入口</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">选择一个娱乐项目开始体验，结算会直接走现有大厅接口。</p>
          </div>

          <div class="p-5">
            <RouterLink
              to="/games/slots"
              class="group relative block overflow-hidden rounded-[1.5rem] border border-primary-200/80 bg-gradient-to-br from-primary-950 via-primary-800 to-accent-900 p-6 text-white shadow-glow transition-all duration-300 hover:-translate-y-1 hover:shadow-glow-lg dark:border-primary-700/40"
            >
              <div class="absolute inset-0 bg-[radial-gradient(circle_at_top_right,rgba(255,255,255,0.16),transparent_35%),radial-gradient(circle_at_bottom_left,rgba(20,184,166,0.22),transparent_40%)]"></div>
              <div class="relative flex flex-col gap-6 lg:flex-row lg:items-center lg:justify-between">
                <div class="space-y-4">
                  <div class="inline-flex items-center gap-2 rounded-full border border-white/10 bg-white/10 px-3 py-1 text-xs font-medium uppercase tracking-[0.24em] text-primary-100">
                    <span class="h-2 w-2 rounded-full bg-emerald-300"></span>
                    Slot Machine
                  </div>
                  <div class="space-y-3">
                    <h3 class="text-2xl font-semibold tracking-wide">幸运老虎机</h3>
                    <p class="max-w-xl text-sm leading-6 text-primary-50/80">
                      经典三轴老虎机，下注与结算全部基于现有 DG 钱包接口。保留原始游戏的热闹氛围，但改成了站内统一的青蓝配色。
                    </p>
                  </div>
                  <div class="flex flex-wrap gap-2 text-xs text-primary-50/80">
                    <span class="rounded-full border border-white/10 bg-white/10 px-3 py-1">后端真实结算</span>
                    <span class="rounded-full border border-white/10 bg-white/10 px-3 py-1">1 条中奖线</span>
                    <span class="rounded-full border border-white/10 bg-white/10 px-3 py-1">即时更新余额</span>
                  </div>
                </div>

                <div class="grid min-w-[220px] grid-cols-3 gap-2 rounded-[1.25rem] border border-white/10 bg-black/20 p-3 backdrop-blur-sm">
                  <div
                    v-for="emoji in slotPreviewSymbols"
                    :key="emoji"
                    class="flex aspect-square items-center justify-center rounded-2xl border border-white/10 bg-white/10 text-3xl shadow-inner shadow-black/20 transition-transform duration-300 group-hover:scale-[1.03]"
                  >
                    {{ emoji }}
                  </div>
                </div>
              </div>

              <div class="relative mt-6 flex items-center justify-between rounded-2xl border border-white/10 bg-white/10 px-4 py-3 text-sm">
                <span class="text-primary-50/80">当前可用范围：{{ slotMinBetText }} - {{ slotMaxBetText }}</span>
                <span class="font-semibold text-white">进入游戏</span>
              </div>
            </RouterLink>
          </div>
        </article>

        <section data-testid="exchange-card" class="card border-primary-100/70 p-5 dark:border-primary-900/30">
          <div class="space-y-1">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">余额兑换</h2>
            <p class="text-sm text-gray-500 dark:text-gray-400">按 1:1 在主余额与 DG 币之间切换。</p>
          </div>

          <div class="mt-5 flex flex-col gap-4">
            <div class="grid gap-3 sm:grid-cols-2">
              <div class="rounded-2xl border border-gray-100 bg-gray-50 px-4 py-3 dark:border-dark-700 dark:bg-dark-800">
                <p class="text-xs text-gray-500 dark:text-gray-400">主余额</p>
                <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ formatMainBalance(hallStatus?.main_balance) }}</p>
              </div>
              <div class="rounded-2xl border border-primary-100 bg-primary-50/70 px-4 py-3 dark:border-primary-900/40 dark:bg-primary-950/20">
                <p class="text-xs text-gray-500 dark:text-gray-400">DG 币</p>
                <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ formatDG(hallStatus?.dg_balance) }}</p>
              </div>
            </div>

            <label>
              <span class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">兑换方向</span>
              <select v-model="exchangeDirection" class="input">
                <option value="balance_to_dg">主余额转 DG</option>
                <option value="dg_to_balance">DG 转主余额</option>
              </select>
            </label>

            <label>
              <span class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300">兑换数量</span>
              <div class="relative">
                <input v-model.number="exchangeAmount" type="number" min="0.01" step="0.01" class="input pr-24 sm:pr-28" />
                <span
                  v-if="exchangeAmountHintText"
                  data-testid="exchange-amount-hint"
                  class="pointer-events-none absolute inset-y-0 right-3 flex items-center text-sm font-medium text-gray-400 dark:text-gray-500"
                >
                  {{ exchangeAmountHintText }}
                </span>
              </div>
              <span
                v-if="exchangeAmountHelperText"
                data-testid="exchange-amount-helper"
                class="mt-1 block text-xs text-gray-500 dark:text-gray-400"
              >
                {{ exchangeAmountHelperText }}
              </span>
            </label>

            <div class="grid gap-2 sm:grid-cols-4">
              <button
                v-for="quickAmount in exchangeQuickAmounts"
                :key="quickAmount"
                type="button"
                class="rounded-xl border border-gray-200 bg-gray-50 px-3 py-2 text-sm font-medium text-gray-600 transition hover:border-primary-300 hover:bg-primary-50 hover:text-primary-700 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300 dark:hover:border-primary-500 dark:hover:bg-primary-900/20 dark:hover:text-primary-300"
                @click="exchangeAmount = quickAmount"
              >
                {{ formatNumber(quickAmount) }}
              </button>
            </div>

            <button class="btn btn-primary w-full" :disabled="exchangeSubmitting" @click="submitExchange">
              {{ exchangeSubmitting ? '兑换中...' : '立即兑换' }}
            </button>
          </div>
        </section>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { gamesAPI, type GameExchangeDirection, type GameHallStatus } from '@/api/games'
import { useAppStore, useAuthStore } from '@/stores'
import { formatNumber } from '@/utils/format'

const slotPreviewSymbols = ['7', '◆', '★', '🔔', '🍇', '🍊']
const exchangeQuickAmounts = [10, 20, 50, 100]

const appStore = useAppStore()
const authStore = useAuthStore()

const hallStatus = ref<GameHallStatus | null>(null)
const errorText = ref('')
const exchangeSubmitting = ref(false)
const exchangeDirection = ref<GameExchangeDirection>('balance_to_dg')
const exchangeAmount = ref(10)
const exchangeTargetUnitLabel = computed(() => (exchangeDirection.value === 'balance_to_dg' ? 'DG币' : '主余额'))
const exchangeAmountHint = computed(() => formatChineseAmountHint(exchangeAmount.value))
const exchangeAmountHintText = computed(() => {
  if (!exchangeAmountHint.value) return ''
  return `${exchangeAmountHint.value}${exchangeTargetUnitLabel.value}`
})
const exchangeAmountHelperText = computed(() => {
  if (!exchangeAmountHint.value) return ''
  return `按 1:1 兑换，预计到账 ${exchangeAmountHint.value}${exchangeTargetUnitLabel.value}`
})
const slotsGame = computed(() => hallStatus.value?.games.find((item) => item.type === 'slots') ?? null)
const slotMinBetText = computed(() => formatDG(slotsGame.value?.min_bet))
const slotMaxBetText = computed(() => formatDG(Math.max(Number(slotsGame.value?.max_bet ?? 0), 100000000)))

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
    await authStore.refreshUser()
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

function formatChineseAmountHint(value: number | string | null | undefined) {
  const amount = Number(value ?? 0)
  if (!Number.isFinite(amount) || amount < 10000) return ''

  if (amount >= 100000000) {
    return `${formatChineseAmountUnit(amount / 100000000)}亿`
  }

  if (amount >= 10000000) {
    const tenMillionValue = Number(formatChineseAmountUnit(amount / 10000000))
    if (tenMillionValue >= 10) {
      return `${formatChineseAmountUnit(amount / 100000000)}亿`
    }
    return `${trimTrailingZeros(String(tenMillionValue))}千万`
  }

  return `${formatChineseAmountUnit(amount / 10000)}万`
}

function formatChineseAmountUnit(value: number) {
  const absValue = Math.abs(value)
  if (!Number.isFinite(absValue) || absValue === 0) return '0'

  const integerDigits = Math.floor(Math.log10(absValue)) + 1
  const fractionDigits = Math.max(0, Math.min(3, 4 - integerDigits))
  return trimTrailingZeros(value.toFixed(fractionDigits))
}

function trimTrailingZeros(value: string) {
  return value.replace(/\.0+$/, '').replace(/(\.\d*[1-9])0+$/, '$1')
}

function toErrorMessage(error: unknown) {
  return (error as { message?: string })?.message || '娱乐大厅加载失败，请稍后再试。'
}
</script>
