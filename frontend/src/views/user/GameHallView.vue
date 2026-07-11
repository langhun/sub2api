<template>
  <AppLayout>
    <div class="mx-auto max-w-6xl space-y-6">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('gameHall.title') }}</h1>
          <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">{{ t('gameHall.description') }}</p>
        </div>
        <button class="btn btn-secondary self-start" type="button" :disabled="store.loading" @click="reload">
          <Icon name="refresh" size="sm" />
          {{ t('common.refresh') }}
        </button>
      </header>

      <div v-if="store.loading && !store.status" class="card flex min-h-56 items-center justify-center">
        <LoadingSpinner />
      </div>
      <div v-else-if="store.error && !store.status" class="card py-12 text-center">
        <Icon name="exclamationCircle" size="xl" class="mx-auto text-red-500" />
        <p class="mt-3 text-sm text-gray-600 dark:text-dark-300">{{ t('gameHall.loadFailed') }}</p>
        <button class="btn btn-primary mt-4" type="button" @click="reload">{{ t('common.retry') }}</button>
      </div>

      <template v-else-if="store.status">
        <section class="grid gap-4 sm:grid-cols-3">
          <div class="card p-5">
            <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-dark-400"><Icon name="dollar" size="sm" />{{ t('gameHall.mainBalance') }}</div>
            <p class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ money(store.status.main_balance) }}</p>
          </div>
          <div class="card p-5">
            <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-dark-400"><Icon name="sparkles" size="sm" />{{ t('gameHall.dgBalance') }}</div>
            <p class="mt-2 text-2xl font-semibold text-primary-600 dark:text-primary-400">{{ money(store.status.dg_balance) }} DG</p>
          </div>
          <div class="card p-5">
            <div class="flex items-center gap-2 text-sm text-gray-500 dark:text-dark-400"><Icon name="database" size="sm" />{{ t('gameHall.jackpot') }}</div>
            <p class="mt-2 text-2xl font-semibold text-amber-600 dark:text-amber-400">{{ money(store.status.jackpot_balance) }} DG</p>
          </div>
        </section>

        <section class="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.72fr)]">
          <div class="card overflow-hidden p-0">
            <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
              <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('gameHall.exchangeTitle') }}</h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('gameHall.exchangeHint') }}</p>
            </div>
            <form class="space-y-4 p-5" @submit.prevent="confirmingExchange = true">
              <div class="grid grid-cols-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
                <button v-for="option in exchangeOptions" :key="option.value" type="button" class="rounded-md px-3 py-2 text-sm font-medium transition-colors" :class="direction === option.value ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-700 dark:text-primary-300' : 'text-gray-500 dark:text-dark-400'" @click="direction = option.value">
                  {{ option.label }}
                </button>
              </div>
              <label class="block">
                <span class="input-label">{{ t('gameHall.amount') }}</span>
                <input v-model.number="exchangeAmount" class="input" type="number" min="0.01" step="0.01" required />
              </label>
              <div class="rounded-lg border border-gray-200 px-4 py-3 text-sm dark:border-dark-700">
                <div class="flex justify-between text-gray-500 dark:text-dark-400"><span>{{ t('gameHall.rate') }}</span><span>1 : 1</span></div>
                <div class="mt-2 flex justify-between font-medium text-gray-900 dark:text-white"><span>{{ t('gameHall.afterExchange') }}</span><span>{{ exchangePreview }}</span></div>
              </div>
              <button class="btn btn-primary w-full" type="submit" :disabled="store.submitting || exchangeAmount <= 0">
                <Icon name="swap" size="sm" />
                {{ store.submitting ? t('common.processing') : t('gameHall.exchangeAction') }}
              </button>
            </form>
          </div>

          <div class="card p-5">
            <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('gameHall.safetyTitle') }}</h2>
            <div class="mt-4 space-y-3 text-sm text-gray-600 dark:text-dark-300">
              <p class="flex gap-2"><Icon name="shield" size="sm" class="mt-0.5 shrink-0 text-primary-500" />{{ t('gameHall.safetyWallet') }}</p>
              <p class="flex gap-2"><Icon name="checkCircle" size="sm" class="mt-0.5 shrink-0 text-primary-500" />{{ t('gameHall.safetySettlement') }}</p>
              <p class="flex gap-2"><Icon name="infoCircle" size="sm" class="mt-0.5 shrink-0 text-amber-500" />{{ t('gameHall.riskNotice') }}</p>
            </div>
          </div>
        </section>

        <section v-if="slots" class="card overflow-hidden p-0">
          <div class="flex flex-col gap-2 border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 class="font-semibold text-gray-900 dark:text-white">{{ slots.name || t('gameHall.slotsTitle') }}</h2>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('gameHall.betRange', { min: money(slots.min_bet), max: money(slots.max_bet) }) }}</p>
            </div>
            <span class="badge badge-success self-start">{{ t('common.enabled') }}</span>
          </div>
          <div class="grid gap-6 p-5 lg:grid-cols-[minmax(0,1fr)_280px]">
            <div class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900">
              <div class="grid grid-cols-3 gap-3" aria-live="polite">
                <div v-for="(symbol, index) in displayedSymbols" :key="`${symbol}-${index}`" class="flex aspect-square min-h-24 items-center justify-center rounded-lg border border-gray-200 bg-white px-2 text-center text-lg font-bold text-gray-800 shadow-sm dark:border-dark-600 dark:bg-dark-800 dark:text-white">
                  {{ symbolLabel(symbol) }}
                </div>
              </div>
              <div v-if="store.lastRound" class="mt-4 rounded-lg px-4 py-3 text-sm" :class="roundClass">
                <div class="flex items-center justify-between gap-3"><strong>{{ outcomeLabel }}</strong><span>{{ signed(store.lastRound.net_amount) }} DG</span></div>
                <p class="mt-1 opacity-80">{{ t('gameHall.payoutSummary', { payout: money(store.lastRound.payout_amount), multiplier: store.lastRound.multiplier.toFixed(2) }) }}</p>
              </div>
            </div>
            <form class="space-y-4" @submit.prevent="submitPlay">
              <label class="block">
                <span class="input-label">{{ t('gameHall.betAmount') }}</span>
                <input v-model.number="betAmount" class="input" type="number" :min="slots.min_bet" :max="slots.max_bet" step="0.01" required />
              </label>
              <div class="grid grid-cols-3 gap-2">
                <button v-for="amount in quickBets" :key="amount" type="button" class="btn btn-secondary px-2" @click="betAmount = amount">{{ money(amount) }}</button>
              </div>
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('gameHall.serverResult') }}</p>
              <button class="btn btn-primary w-full" type="submit" :disabled="store.submitting || !validBet">
                <Icon name="play" size="sm" />
                {{ store.submitting ? t('common.processing') : t('gameHall.playAction') }}
              </button>
            </form>
          </div>
        </section>

        <div v-else class="card py-12 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('gameHall.noGames') }}</div>

        <section class="card overflow-hidden p-0">
          <div class="grid grid-cols-2 border-b border-gray-100 p-1 dark:border-dark-700"><button type="button" class="px-3 py-2 text-sm font-medium" :class="historyTab === 'transactions' ? 'text-primary-600 dark:text-primary-400' : 'text-gray-500'" @click="changeHistoryTab('transactions')">{{ t('gameHall.transactions') }}</button><button type="button" class="px-3 py-2 text-sm font-medium" :class="historyTab === 'rounds' ? 'text-primary-600 dark:text-primary-400' : 'text-gray-500'" @click="changeHistoryTab('rounds')">{{ t('gameHall.rounds') }}</button></div>
          <div v-if="historyLoading" class="flex min-h-40 items-center justify-center"><LoadingSpinner /></div>
          <div v-else-if="historyError" class="py-10 text-center"><p class="text-sm text-red-600 dark:text-red-400">{{ historyError }}</p><button class="btn btn-secondary mt-3" @click="loadHistory">{{ t('common.retry') }}</button></div>
          <div v-else-if="historyTab === 'transactions' && transactions.length" class="divide-y divide-gray-100 dark:divide-dark-700"><div v-for="item in transactions" :key="item.id" class="flex items-center justify-between px-5 py-3 text-sm"><div><p class="font-medium text-gray-900 dark:text-white">{{ t(`gameHall.transactionTypes.${item.tx_type}`, item.tx_type) }}</p><p class="mt-1 text-xs text-gray-500">{{ dateTime(item.created_at) }}</p></div><strong :class="item.amount >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">{{ signed(item.amount) }} DG</strong></div></div>
          <div v-else-if="historyTab === 'rounds' && rounds.length" class="divide-y divide-gray-100 dark:divide-dark-700"><div v-for="round in rounds" :key="round.id" class="flex items-center justify-between px-5 py-3 text-sm"><div><p class="font-medium text-gray-900 dark:text-white">#{{ round.id }} · {{ t('gameHall.slotsTitle') }}</p><p class="mt-1 text-xs text-gray-500">{{ dateTime(round.created_at) }} · {{ t('gameHall.betValue', { value: money(round.bet_amount) }) }}</p></div><strong :class="round.net_amount >= 0 ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'">{{ signed(round.net_amount) }} DG</strong></div></div>
          <div v-else class="py-10 text-center text-sm text-gray-500 dark:text-dark-400">{{ t('gameHall.emptyHistory') }}</div>
          <Pagination v-if="historyTotal > historyPageSize" :page="historyPage" :page-size="historyPageSize" :total="historyTotal" :show-page-size-selector="false" @update:page="changeHistoryPage" />
        </section>
      </template>
      <ConfirmDialog :show="confirmingExchange" :title="t('gameHall.confirmExchangeTitle')" :message="t('gameHall.confirmExchangeMessage', { amount: money(exchangeAmount), direction: direction === 'balance_to_dg' ? t('gameHall.toDG') : t('gameHall.toMain') })" @confirm="submitExchange" @cancel="confirmingExchange = false" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Pagination from '@/components/common/Pagination.vue'
import { useAppStore } from '@/stores/app'
import { useGameHallStore } from '@/stores/gameHall'
import type { GameExchangeResult } from '@/api/gameHall'
import { getGameRounds, getGameTransactions, type GameRound, type GameWalletTransaction } from '@/api/gameHall'

const { t } = useI18n()
const appStore = useAppStore()
const store = useGameHallStore()
const direction = ref<GameExchangeResult['direction']>('balance_to_dg')
const exchangeAmount = ref(10)
const betAmount = ref(1)
const confirmingExchange = ref(false)
const historyTab = ref<'transactions' | 'rounds'>('transactions')
const transactions = ref<GameWalletTransaction[]>([]); const rounds = ref<GameRound[]>([]); const historyLoading = ref(false); const historyError = ref(''); const historyPage = ref(1); const historyTotal = ref(0); const historyPageSize = 10

const exchangeOptions = computed(() => [
  { value: 'balance_to_dg' as const, label: t('gameHall.toDG') },
  { value: 'dg_to_balance' as const, label: t('gameHall.toMain') },
])
const slots = computed(() => store.enabledGames.find((game) => game.type === 'slots'))
const displayedSymbols = computed(() => store.lastRound?.symbols?.length ? store.lastRound.symbols : ['STAR', 'SEVEN', 'DIAMOND'])
const validBet = computed(() => !!slots.value && betAmount.value >= slots.value.min_bet && betAmount.value <= slots.value.max_bet && betAmount.value <= (store.status?.dg_balance ?? 0))
const quickBets = computed(() => {
  if (!slots.value) return []
  const values = [slots.value.min_bet, Math.max(slots.value.min_bet, Math.min(5, slots.value.max_bet)), slots.value.max_bet]
  return [...new Set(values)]
})
const exchangePreview = computed(() => {
  if (!store.status) return '-'
  const amount = Math.max(0, Number(exchangeAmount.value) || 0)
  return direction.value === 'balance_to_dg'
    ? `${money(Math.max(0, store.status.main_balance - amount))} / ${money(store.status.dg_balance + amount)} DG`
    : `${money(store.status.main_balance + amount)} / ${money(Math.max(0, store.status.dg_balance - amount))} DG`
})
const roundClass = computed(() => store.lastRound?.net_amount && store.lastRound.net_amount > 0
  ? 'bg-green-50 text-green-700 dark:bg-green-900/20 dark:text-green-300'
  : store.lastRound?.net_amount === 0
    ? 'bg-gray-100 text-gray-700 dark:bg-dark-800 dark:text-dark-200'
    : 'bg-red-50 text-red-700 dark:bg-red-900/20 dark:text-red-300')
const outcomeLabel = computed(() => store.lastRound?.net_amount && store.lastRound.net_amount > 0 ? t('gameHall.win') : store.lastRound?.net_amount === 0 ? t('gameHall.push') : t('gameHall.loss'))

function money(value: number): string { return Number(value || 0).toFixed(2) }
function signed(value: number): string { return `${value > 0 ? '+' : ''}${money(value)}` }
function dateTime(value: string): string { return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value)) }
function symbolLabel(symbol: string): string { return t(`gameHall.symbols.${symbol.toLowerCase()}`, symbol.replace(/_/g, ' ')) }

async function reload(): Promise<void> {
  try { await store.refresh() } catch { /* visible error state is stored */ }
}
async function submitExchange(): Promise<void> {
	confirmingExchange.value = false
  try {
    await store.exchange(direction.value, exchangeAmount.value)
    appStore.showSuccess(t('gameHall.exchangeSuccess'))
  } catch { appStore.showError(t('gameHall.exchangeFailed')) }
}
async function submitPlay(): Promise<void> {
  if (!slots.value) return
  try { await store.play(slots.value.type, betAmount.value); await loadHistory() } catch { appStore.showError(store.error || t('gameHall.playFailed')) }
}

async function loadHistory(): Promise<void> { historyLoading.value = true; historyError.value = ''; try { const result = historyTab.value === 'transactions' ? await getGameTransactions(historyPage.value, historyPageSize) : await getGameRounds(historyPage.value, historyPageSize); if (historyTab.value === 'transactions') transactions.value = result.items as GameWalletTransaction[]; else rounds.value = result.items as GameRound[]; historyTotal.value = result.total } catch { historyError.value = store.error || t('gameHall.historyFailed') } finally { historyLoading.value = false } }
function changeHistoryTab(tab: 'transactions' | 'rounds') { historyTab.value = tab; historyPage.value = 1; void loadHistory() }
function changeHistoryPage(page: number) { historyPage.value = page; void loadHistory() }

onMounted(async () => { await Promise.all([reload(), loadHistory()]) })
</script>
