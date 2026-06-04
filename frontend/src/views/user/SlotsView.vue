<template>
  <AppLayout>
    <div class="slots-page mx-auto max-w-5xl space-y-5">
      <header class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <p class="text-sm font-medium text-[var(--muted-foreground)]">娱乐大厅</p>
          <h1 class="text-2xl font-semibold text-[var(--foreground)]">老虎机</h1>
        </div>
        <div class="flex flex-wrap gap-2">
          <RouterLink class="btn btn-secondary" to="/games">返回大厅</RouterLink>
          <button class="btn btn-secondary" type="button" :disabled="loading || spinning" @click="loadHall">
            {{ loading ? '刷新中' : '刷新余额' }}
          </button>
        </div>
      </header>

      <p v-if="error" class="rounded-md border border-[var(--destructive)]/30 bg-[var(--destructive)]/10 px-4 py-3 text-sm text-[var(--destructive)]">
        {{ error }}
      </p>

      <section class="grid gap-3 md:grid-cols-3">
        <div class="summary-cell">
          <span>娱乐余额</span>
          <strong>{{ loading ? '加载中' : formatMoney(gameHall?.balance) }}</strong>
        </div>
        <div class="summary-cell">
          <span>投注范围</span>
          <strong>{{ betRangeText }}</strong>
        </div>
        <div class="summary-cell">
          <span>最高倍率</span>
          <strong>{{ maxMultiplierText }}</strong>
        </div>
      </section>

      <main class="slot-panel">
        <div class="slot-machine" :class="{ spinning }" data-testid="slot-machine" aria-live="polite">
          <div
            v-for="(_, index) in displaySymbols"
            :key="index"
            class="slot-reel"
            :class="{ 'is-spinning': spinning }"
            :data-testid="`slot-reel-${index}`"
          >
            <div class="reel-track">
              <span v-for="symbol in reelWindow(index)" :key="`${index}-${symbol}`" class="reel-symbol">
                {{ slotSymbolText(symbol) }}
              </span>
            </div>
          </div>
        </div>

        <div class="grid gap-4 lg:grid-cols-[minmax(0,1fr)_18rem]">
          <section class="result-area">
            <div v-if="slotResult" class="game-result" data-testid="slots-result">
              <div class="flex flex-wrap items-center gap-2">
                <span class="status-pill" :class="outcomeClass(slotResult.outcome)">{{ outcomeText(slotResult.outcome) }}</span>
                <strong>{{ slotResult.message }}</strong>
              </div>
              <div class="mt-3 grid gap-1 text-sm text-[var(--muted-foreground)] sm:grid-cols-3">
                <span>投注：{{ formatMoney(slotResult.bet_amount) }}</span>
                <span>返奖：{{ formatMoney(slotResult.payout_amount) }}</span>
                <span>余额：{{ formatMoney(slotResult.balance_after) }}</span>
              </div>
            </div>
            <div v-else class="result-placeholder">
              {{ spinning ? '滚动中，等待结算结果。' : '设置投注金额后拉杆。' }}
            </div>
          </section>

          <section class="control-panel">
            <label class="block text-sm font-medium text-[var(--foreground)]" for="slot-bet-amount">投注金额</label>
            <input
              id="slot-bet-amount"
              v-model.number="slotBetAmount"
              class="input"
              data-testid="slot-bet-amount"
              type="number"
              inputmode="decimal"
              :min="slotGame?.min_bet ?? 0.01"
              :max="slotGame?.max_bet ?? 100000000"
              step="0.01"
              :disabled="spinning"
            >
            <p class="min-h-10 text-sm text-[var(--muted-foreground)]">{{ slotBetHint }}</p>
            <button
              class="btn btn-primary w-full"
              type="button"
              data-testid="play-slots"
              :disabled="!canPlaySlots"
              @click="playSlots"
            >
              {{ spinning ? '滚动中' : '拉杆' }}
            </button>
          </section>
        </div>
      </main>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { gamesAPI, type GameHallStatus, type GameInfo, type GameOutcome, type GamePlayResult } from '@/api/games'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores'
import { formatDualDisplayAmount } from '@/utils/format'

const SPIN_MS = 900
const SYMBOLS = [
  { key: 'cherry', label: '樱桃' },
  { key: 'lemon', label: '柠檬' },
  { key: 'orange', label: '橙子' },
  { key: 'grape', label: '葡萄' },
  { key: 'bell', label: '铃铛' },
  { key: 'star', label: '星星' },
  { key: 'diamond', label: '钻石' },
  { key: '7', label: '7' },
]
const SYMBOL_KEYS = SYMBOLS.map((symbol) => symbol.key)
const DEFAULT_SYMBOLS = ['7', 'bell', 'star']

const appStore = useAppStore()

const gameHall = ref<GameHallStatus | null>(null)
const slotResult = ref<GamePlayResult | null>(null)
const displaySymbols = ref<string[]>([...DEFAULT_SYMBOLS])
const slotBetAmount = ref(100)
const loading = ref(false)
const spinning = ref(false)
const error = ref('')
let spinTimer: number | null = null
let spinTick = 0

const slotGame = computed(() => gameHall.value?.games.find((game) => game.type === 'slots') ?? null)
const maxMultiplier = computed(() => Math.max(...(slotGame.value?.multipliers ?? [0])))
const maxMultiplierText = computed(() => slotGame.value ? `${formatMultiplier(maxMultiplier.value)}x` : '-')
const betRangeText = computed(() => {
  if (!slotGame.value) return '-'
  return `${formatMoney(slotGame.value.min_bet)} - ${formatMoney(slotGame.value.max_bet)}`
})
const canPlaySlots = computed(() => {
  const bet = Number(slotBetAmount.value)
  const game = slotGame.value
  if (!game) return false
  return !spinning.value &&
    Number.isFinite(bet) &&
    bet >= game.min_bet &&
    bet <= game.max_bet
})
const slotBetHint = computed(() => {
  if (loading.value) return '正在加载老虎机状态。'
  const game = slotGame.value
  if (!game) return '老虎机暂未开放。'
  const bet = Number(slotBetAmount.value)
  if (!Number.isFinite(bet)) return '请输入有效投注金额。'
  if (bet < game.min_bet) return `最低投注 ${formatMoney(game.min_bet)}。`
  if (bet > game.max_bet) return `最高投注 ${formatMoney(game.max_bet)}。`
  return `可用倍率：${formatMultipliers(game)}`
})

onMounted(() => {
  void loadHall()
})

onBeforeUnmount(() => {
  stopRolling()
})

async function loadHall() {
  loading.value = true
  error.value = ''
  try {
    gameHall.value = await gamesAPI.getHall()
  } catch (value) {
    gameHall.value = null
    error.value = readableError(value)
    appStore.showError(error.value)
  } finally {
    loading.value = false
  }
}

async function playSlots() {
  if (!canPlaySlots.value) return
  spinning.value = true
  slotResult.value = null
  error.value = ''
  startRolling()

  try {
    const [result] = await Promise.all([
      gamesAPI.play('slots', Number(slotBetAmount.value)),
      delay(SPIN_MS),
    ])
    slotResult.value = result
    displaySymbols.value = normalizeSymbols(result.symbols)
    appStore.showSuccess(result.message || '老虎机已结算')
    await loadHall()
  } catch (value) {
    error.value = readableError(value)
    appStore.showError(error.value)
  } finally {
    stopRolling()
    spinning.value = false
  }
}

function startRolling() {
  stopRolling()
  spinTimer = window.setInterval(() => {
    displaySymbols.value = displaySymbols.value.map((_, index) => {
      const symbolIndex = (spinTick + index * 2) % SYMBOL_KEYS.length
      return SYMBOL_KEYS[symbolIndex]
    })
    spinTick += 1
  }, 80)
}

function stopRolling() {
  if (spinTimer !== null) {
    window.clearInterval(spinTimer)
    spinTimer = null
  }
}

function reelWindow(index: number) {
  const current = displaySymbols.value[index] ?? '?'
  const currentIndex = SYMBOL_KEYS.indexOf(current)
  if (currentIndex === -1) return ['?', current, '?']
  return [
    SYMBOL_KEYS[(currentIndex + SYMBOL_KEYS.length - 1) % SYMBOL_KEYS.length],
    current,
    SYMBOL_KEYS[(currentIndex + 1) % SYMBOL_KEYS.length],
  ]
}

function normalizeSymbols(symbols: string[] | undefined) {
  if (!symbols?.length) return [...DEFAULT_SYMBOLS]
  const normalized = symbols.slice(0, 3)
  while (normalized.length < 3) {
    normalized.push(DEFAULT_SYMBOLS[normalized.length])
  }
  return normalized
}

function delay(ms: number) {
  return new Promise((resolve) => window.setTimeout(resolve, ms))
}

function slotSymbolText(symbol: string) {
  return SYMBOLS.find((item) => item.key === symbol)?.label ?? symbol
}

function formatMoney(value: string | number | null | undefined) {
  const amount = Number(value ?? 0)
  return `${formatDualDisplayAmount(Number.isFinite(amount) ? amount : 0).display} DG`
}

function formatMultipliers(game: GameInfo) {
  return game.multipliers.map((value) => `${formatMultiplier(value)}x`).join(' / ')
}

function formatMultiplier(value: number) {
  return String(value).replace(/\.0+$/, '')
}

function outcomeText(outcome: GameOutcome) {
  if (outcome === 'win') return '已中奖'
  if (outcome === 'draw') return '保本'
  return '未中奖'
}

function outcomeClass(outcome: GameOutcome) {
  if (outcome === 'win') return 'status-win'
  if (outcome === 'draw') return 'status-pending'
  return 'status-lose'
}

function readableError(value: unknown) {
  const errorValue = value as { message?: string; reason?: string }
  if (errorValue?.reason === 'BANK_INSUFFICIENT_BALANCE') return 'DG 币余额不足，无法开始游戏。'
  return errorValue?.message || '请求失败，请稍后重试。'
}
</script>

<style scoped>
.slots-page {
  padding-bottom: 2rem;
}

.summary-cell,
.slot-panel,
.game-result,
.control-panel,
.result-placeholder {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.05);
}

.summary-cell {
  min-height: 5.25rem;
  padding: 1rem;
}

.summary-cell span {
  display: block;
  font-size: 0.875rem;
  color: var(--muted-foreground);
}

.summary-cell strong {
  display: block;
  margin-top: 0.5rem;
  overflow-wrap: anywhere;
  font-size: 1.125rem;
  color: var(--foreground);
}

.slot-panel {
  display: grid;
  gap: 1.25rem;
  padding: 1.25rem;
}

.slot-machine {
  display: grid;
  grid-template-columns: repeat(3, minmax(5rem, 1fr));
  gap: 0.75rem;
  min-height: 7rem;
}

.slot-reel {
  position: relative;
  height: 6.25rem;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: linear-gradient(180deg, var(--muted), var(--card) 44%, var(--muted));
}

.slot-reel::before,
.slot-reel::after {
  content: '';
  position: absolute;
  z-index: 1;
  right: 0;
  left: 0;
  height: 1.25rem;
  pointer-events: none;
}

.slot-reel::before {
  top: 0;
  background: linear-gradient(180deg, rgb(0 0 0 / 0.16), transparent);
}

.slot-reel::after {
  bottom: 0;
  background: linear-gradient(0deg, rgb(0 0 0 / 0.16), transparent);
}

.reel-track {
  display: grid;
  height: 18.75rem;
  transform: translateY(-6.25rem);
}

.is-spinning .reel-track {
  animation: reel-roll 0.2s linear infinite;
}

.reel-symbol {
  display: flex;
  height: 6.25rem;
  align-items: center;
  justify-content: center;
  color: var(--foreground);
  font-size: clamp(1.25rem, 3vw, 2rem);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.control-panel,
.result-placeholder,
.game-result {
  padding: 1rem;
}

.control-panel {
  display: grid;
  gap: 0.75rem;
  align-content: start;
}

.result-placeholder {
  min-height: 7rem;
  color: var(--muted-foreground);
}

.status-pill {
  flex-shrink: 0;
  border-radius: 6px;
  padding: 0.25rem 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
}

.status-pending {
  background: var(--muted);
  color: var(--muted-foreground);
}

.status-win {
  background: rgb(16 185 129 / 0.14);
  color: rgb(4 120 87);
}

.status-lose {
  background: rgb(148 163 184 / 0.16);
  color: var(--muted-foreground);
}

.dark .status-win {
  color: rgb(110 231 183);
}

@keyframes reel-roll {
  from {
    transform: translateY(-12.5rem);
  }

  to {
    transform: translateY(0);
  }
}

@media (max-width: 640px) {
  .slot-machine {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
</style>
