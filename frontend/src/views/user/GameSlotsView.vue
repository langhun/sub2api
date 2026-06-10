<template>
  <AppLayout>
    <div class="slots-page">
      <header class="slots-page__header">
        <div class="slots-page__header-row">
          <div class="slots-page__heading">
            <RouterLink
              to="/games"
              class="slots-page__back inline-flex items-center gap-2 text-sm font-medium text-primary-600 transition hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
            >
              <span aria-hidden="true">←</span>
              返回娱乐大厅
            </RouterLink>

            <div class="slots-page__title-block">
              <p class="slots-page__eyebrow text-xs font-semibold uppercase tracking-[0.28em] text-gray-400 dark:text-gray-500">Lucky Slots</p>
              <h1 class="slots-title slots-page__title text-3xl font-semibold tracking-[-0.04em] text-gray-950 dark:text-white sm:text-[3.1rem]">
                <span class="slots-title__icon" aria-hidden="true">🎰</span>
                <span>幸运老虎机</span>
              </h1>
            </div>
          </div>

        </div>
      </header>

      <p
        v-if="errorText"
        class="rounded-2xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-600 dark:border-red-900/50 dark:bg-red-950/40 dark:text-red-300"
      >
        {{ errorText }}
      </p>

      <section class="slots-layout">
        <article class="machine-panel">
          <div class="machine-stats">
            <div class="machine-stats__item machine-stats__item--centered">
              <span>DG 币</span>
              <strong>{{ formatPlainAmount(hallStatus?.dg_balance) }}</strong>
            </div>
            <div class="machine-stats__item machine-stats__item--centered">
              <span>胜率</span>
              <strong>{{ winRateDisplay }}</strong>
            </div>
            <div class="machine-stats__item machine-stats__item--centered">
              <span>本场盈亏</span>
              <strong :class="sessionNetClass">{{ sessionNetValueDisplay }}</strong>
            </div>
          </div>

          <div class="slot-machine">
            <div
              ref="reelFrameRef"
              class="slot-machine__frame"
              :class="{ 'slot-machine__frame--win': isWinning }"
            >
              <div class="slot-machine__scanline"></div>
              <div class="slot-machine__reels">
                <div
                  v-for="(column, columnIndex) in reelColumns"
                  :key="`reel-${columnIndex}`"
                  class="slot-machine__reel"
                  :class="{ 'slot-machine__reel--spinning': isSpinning }"
                >
                  <div
                    :data-testid="`slots-reel-strip-${columnIndex}`"
                    class="slot-machine__strip"
                    :class="{ 'slot-machine__strip--spinning': isSpinning }"
                    :style="{ transform: `translateY(${column.offsetY}px)` }"
                  >
                    <div
                      v-for="(symbol, rowIndex) in column.symbols"
                      :key="`symbol-${columnIndex}-${rowIndex}-${symbol.id}-${rowIndex === reelCenterIndex ? 'center' : 'buffer'}`"
                      class="slot-machine__symbol"
                      :class="{
                        'slot-machine__symbol--center': rowIndex === reelCenterIndex,
                        'slot-machine__symbol--landed': column.landedAt > 0 && rowIndex >= reelBufferCount && rowIndex <= reelBufferCount + 2,
                        'slot-machine__symbol--winner': isWinning && rowIndex === reelCenterIndex && resultSymbols.length === 3,
                      }"
                    >
                      <span>{{ symbol.emoji }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="machine-actions">
            <div class="machine-actions__group">
              <button
                v-for="bet in betOptions"
                :key="bet"
                :data-testid="`slots-bet-${bet}`"
                type="button"
                class="machine-actions__bet"
                :class="{ 'machine-actions__bet--active': selectedBet === bet }"
                @click="selectPresetBet(bet)"
              >
                {{ formatCompactBet(bet) }}
              </button>
            </div>

            <div class="machine-actions__footer">
              <label class="machine-actions__custom">
                <span>自定义 DG 投注</span>
                <div class="machine-actions__custom-input">
                  <input
                    v-model="customBetInput"
                    data-testid="slots-custom-bet"
                    type="number"
                    inputmode="numeric"
                    :min="betBounds.minBet"
                    :max="betBounds.maxBet ?? undefined"
                    :placeholder="`${betBounds.minBet} 起投，最高 1亿`"
                    :disabled="isSpinning"
                  >
                  <span
                    v-if="customBetChineseHint"
                    data-testid="slots-custom-bet-hint"
                    class="machine-actions__custom-hint"
                  >
                    {{ customBetChineseHint }}
                  </span>
                </div>
              </label>

              <button
                data-testid="slots-spin-button"
                class="machine-actions__spin"
                :disabled="isSpinning || !hallStatus || !slotsGame"
                @click="spinSlots(1)"
              >
                {{ spinButtonText }}
              </button>

              <div class="machine-actions__batch">
                <button
                  data-testid="slots-spin-5-button"
                  type="button"
                  :disabled="isSpinning || !hallStatus || !slotsGame"
                  @click="spinSlots(5)"
                >
                  5 连抽
                </button>
                <button
                  data-testid="slots-spin-10-button"
                  type="button"
                  :disabled="isSpinning || !hallStatus || !slotsGame"
                  @click="spinSlots(10)"
                >
                  10 连抽
                </button>
              </div>
            </div>
          </div>
        </article>

        <aside class="info-column">
          <section class="info-card info-card--history">
            <div class="info-card__titlebar">
              <div class="flex items-center gap-2">
                <span class="text-lg">📜</span>
                <h3 class="text-xl font-semibold tracking-[-0.03em] text-gray-950 dark:text-white">历史记录</h3>
              </div>
              <span class="text-xl font-semibold italic text-gray-950 dark:text-white">{{ spinHistoryLimit }}</span>
            </div>

            <div class="history-table">
              <div v-if="!spinHistory.length" class="history-table__empty">
                还没有历史结果，先转一局看看。
              </div>

              <div v-for="entry in spinHistory" :key="entry.id" class="history-table__row">
                <span class="history-table__time">{{ formatHistoryTime(entry.createdAt) }}</span>
                <div class="history-table__symbols">
                  <span
                    v-for="(symbol, index) in entry.symbols"
                    :key="`${entry.id}-${index}`"
                    class="history-table__symbol"
                  >
                    {{ mapSymbol(symbol).emoji }}
                  </span>
                </div>
                <span class="history-table__amount" :class="historyNetClass(entry.netAmount)">{{ formatSignedAmount(entry.netAmount) }}</span>
              </div>
            </div>
          </section>

          <section class="info-card info-card--tip">
            <div class="flex items-center gap-2">
              <span class="text-lg">💡</span>
              <h3 class="text-xl font-semibold tracking-[-0.03em] text-gray-950 dark:text-white">友好模式</h3>
            </div>
            <p class="mt-4 text-sm leading-7 text-gray-500 dark:text-gray-400">
              自定义投注和连抽都会逐局真实结算，建议先用 10 / 100 / 1000 试手感，再根据 DG 币余额逐步切到 1万 / 10万 / 100万。
            </p>
          </section>

          <section class="info-card info-card--odds">
            <button
              type="button"
              class="info-card__titlebar info-card__titlebar--button"
              data-testid="slots-odds-toggle"
              :aria-expanded="isOddsExpanded"
              aria-controls="slots-odds-table"
              @click="isOddsExpanded = !isOddsExpanded"
            >
              <span class="flex items-center gap-2">
                <span class="text-lg">🧾</span>
                <span class="text-xl font-semibold tracking-[-0.03em] text-gray-950 dark:text-white">赔率表</span>
              </span>
              <span class="odds-toggle">
                {{ isOddsExpanded ? '收起' : '展开' }}
                <span class="odds-toggle__chevron" :class="{ 'odds-toggle__chevron--open': isOddsExpanded }">⌄</span>
              </span>
            </button>

            <div v-if="isOddsExpanded" id="slots-odds-table" class="odds-table">
              <div v-for="item in payoutTable" :key="item.id" class="odds-table__row">
                <div class="odds-table__pattern">
                  <span>{{ item.emoji }}</span>
                  <span v-if="item.label" class="odds-table__label">{{ item.label }}</span>
                </div>
                <span class="odds-table__value">{{ item.multiplier }}</span>
              </div>
            </div>
          </section>
        </aside>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { gamesAPI, type GameHallStatus, type GameInfo, type GamePlayResult } from '@/api/games'
import { useAppStore } from '@/stores'
import { formatNumber } from '@/utils/format'

type SymbolMeta = {
  id: string
  emoji: string
}

type SpinHistoryEntry = {
  id: number
  outcome: GamePlayResult['outcome']
  netAmount: number
  multiplier: number
  symbols: string[]
  createdAt: number
}

type ReelColumnState = {
  symbols: SymbolMeta[]
  offsetY: number
  landedAt: number
}

type ReelEngineState = {
  phase: 'idle' | 'spin' | 'decel' | 'done'
  position: number
  decelStartAt: number
  decelStartPos: number
}

const appStore = useAppStore()

const symbolCatalog: SymbolMeta[] = [
  { id: 'cherry', emoji: '🍒' },
  { id: 'lemon', emoji: '🍋' },
  { id: 'orange', emoji: '🍊' },
  { id: 'grape', emoji: '🍇' },
  { id: 'bell', emoji: '🔔' },
  { id: 'star', emoji: '⭐' },
  { id: 'diamond', emoji: '💎' },
  { id: 'seven', emoji: '7️⃣' },
]
const fallbackSymbol = symbolCatalog[0]
const symbolMap = new Map(symbolCatalog.map((item) => [item.id, item]))
const visibleSlotsPerReel = 3
const reelBufferCount = 8
const reelGap = ref(8)
const reelCenterIndex = reelBufferCount + 1
const reelCellSize = ref(64)
const reelFrameRef = ref<HTMLElement | null>(null)
const reelSpinDurations = [1000, 1500, 2000] as const
const reelDecelDurationMs = 500
const reelSpinSpeedPx = 18
const spinHistoryLimit = 10

const hallStatus = ref<GameHallStatus | null>(null)
const slotsGame = ref<GameInfo | null>(null)
const errorText = ref('')
const isSpinning = ref(false)
const selectedBet = ref(10)
const customBetInput = ref('')
const reelColumns = ref<ReelColumnState[]>(createInitialReelColumns())
const resultSymbols = ref<string[]>([])
const lastResult = ref<GamePlayResult | null>(null)
const reelAnimationFrame = ref<number | null>(null)
const reelEngine = ref<ReelEngineState[]>(createInitialReelEngine())
const reelStartTime = ref(0)
const pendingResultSymbols = ref<string[] | null>(null)
const spinHistory = ref<SpinHistoryEntry[]>([])
const isOddsExpanded = ref(false)
const spinBatchTotal = ref(0)
const spinBatchIndex = ref(0)
let reelCompletionResolver: (() => void) | null = null
let reelCompletionPromise: Promise<void> | null = null
let spinHistorySeed = 0

const betBounds = computed(() => getBetBounds(slotsGame.value))
const betOptions = computed(() => {
  const { minBet, maxBet } = betBounds.value
  const options = [10, 100, 1000, 10000, 100000, 1000000]

  return Array.from(new Set(options))
    .filter((value) => value >= minBet && (maxBet == null || value <= maxBet))
    .sort((a, b) => a - b)
})
const customBetValue = computed(() => {
  const amount = Number(customBetInput.value)
  if (!Number.isFinite(amount) || amount <= 0) return null
  return Math.floor(amount)
})
const customBetChineseHint = computed(() => formatChineseAmountHint(customBetValue.value))
const activeBetAmount = computed(() => normalizeBet(customBetValue.value ?? selectedBet.value, slotsGame.value))
const isWinning = computed(() => (lastResult.value?.net_amount ?? 0) > 0)
const spinButtonText = computed(() => {
  if (!isSpinning.value) return '🎰 开始旋转'
  if (spinBatchTotal.value > 1) return `连抽中 ${spinBatchIndex.value}/${spinBatchTotal.value}`
  return '旋转中'
})
const sessionSummary = computed(() => {
  const history = spinHistory.value
  const totalSpins = history.length
  const wins = history.filter((entry) => entry.outcome === 'win').length
  const totalNet = history.reduce((sum, entry) => sum + entry.netAmount, 0)

  return {
    totalSpins,
    wins,
    totalNet,
    winRate: totalSpins > 0 ? Math.round((wins / totalSpins) * 100) : 0,
  }
})
const winRateDisplay = computed(() => (sessionSummary.value.totalSpins ? `${sessionSummary.value.winRate}%` : '0%'))
const sessionNetClass = computed(() => getAmountClass(sessionSummary.value.totalNet))
const sessionNetValueDisplay = computed(() => (sessionSummary.value.totalSpins ? formatSignedAmount(sessionSummary.value.totalNet) : '0.00'))
const payoutTable = computed(() => [
  { id: 'seven', emoji: '7️⃣ 7️⃣ 7️⃣', multiplier: '50x', label: '' },
  { id: 'diamond', emoji: '💎 💎 💎', multiplier: '30x', label: '' },
  { id: 'star', emoji: '⭐ ⭐ ⭐', multiplier: '18x', label: '' },
  { id: 'bell', emoji: '🔔 🔔 🔔', multiplier: '12x', label: '' },
  { id: 'grape', emoji: '🍇 🍇 🍇', multiplier: '8x', label: '' },
  { id: 'lemon-orange', emoji: '🍋 / 🍊', multiplier: '5x', label: '三连' },
  { id: 'cherry', emoji: '🍒 🍒 🍒', multiplier: '3x', label: '' },
])

onMounted(() => {
  syncReelMetrics()
  window.addEventListener('resize', syncReelMetrics)
  void loadHall()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', syncReelMetrics)
  clearReelAnimation()
})

async function loadHall() {
  errorText.value = ''
  try {
    const hall = await gamesAPI.getHall()
    hallStatus.value = hall
    slotsGame.value = hall.games.find((item) => item.type === 'slots') ?? null
    if (slotsGame.value) {
      selectedBet.value = normalizeBet(selectedBet.value, slotsGame.value)
    }
    await nextTick()
    syncReelMetrics()
    applyResultGrid([])
  } catch (error) {
    errorText.value = toErrorMessage(error)
    appStore.showError(errorText.value)
  }
}

async function spinSlots(rounds = 1) {
  if (isSpinning.value || !slotsGame.value || !hallStatus.value) return

  const totalRounds = normalizeSpinRounds(rounds)
  const betAmount = activeBetAmount.value
  isSpinning.value = true
  errorText.value = ''
  spinBatchTotal.value = totalRounds
  spinBatchIndex.value = 0

  try {
    for (let round = 0; round < totalRounds; round += 1) {
      spinBatchIndex.value = round + 1
      lastResult.value = null
      resultSymbols.value = []
      pendingResultSymbols.value = null
      await startSpinAnimation()

      const resultState = await gamesAPI.play('slots', betAmount)

      pendingResultSymbols.value = resultState.symbols ?? []
      await waitForReelCompletion()
      hallStatus.value.dg_balance = resultState.dg_balance_after
      hallStatus.value.jackpot_balance = resultState.jackpot_balance
      lastResult.value = resultState
      resultSymbols.value = resultState.symbols ?? []
      recordSpin(resultState)
    }
  } catch (error) {
    clearReelAnimation()
    reelCompletionResolver?.()
    reelCompletionResolver = null
    applyResultGrid([])
    errorText.value = toErrorMessage(error)
    appStore.showError(errorText.value)
  } finally {
    isSpinning.value = false
    pendingResultSymbols.value = null
    spinBatchTotal.value = 0
    spinBatchIndex.value = 0
  }
}

async function startSpinAnimation() {
  clearReelAnimation()
  reelColumns.value = Array.from({ length: 3 }, () => ({
    symbols: createSpinningColumn(),
    offsetY: getReelStartOffset(),
    landedAt: 0,
  }))
  reelEngine.value = Array.from({ length: 3 }, () => ({
    phase: 'spin',
    position: getReelStartOffset(),
    decelStartAt: 0,
    decelStartPos: 0,
  }))
  reelCompletionPromise = new Promise((resolve) => {
    reelCompletionResolver = resolve
  })
  reelStartTime.value = performance.now()
  await nextTick()
  syncReelMetrics()
  reelAnimationFrame.value = window.requestAnimationFrame(tickReels)
}

function clearReelAnimation() {
  if (reelAnimationFrame.value != null) {
    window.cancelAnimationFrame(reelAnimationFrame.value)
    reelAnimationFrame.value = null
  }
}

function applyResultGrid(symbolIds: string[]) {
  const centerRow = normalizeResultSymbols(symbolIds)
  reelColumns.value = centerRow.map((centerSymbol, columnIndex) => ({
    ...createResultColumn(centerSymbol),
    landedAt: reelColumns.value[columnIndex]?.landedAt ?? Date.now(),
  }))
}

function normalizeResultSymbols(symbolIds: string[]) {
  const normalized = symbolIds.slice(0, 3)
  while (normalized.length < 3) {
    normalized.push(randomSymbol().id)
  }
  return normalized
}

function createInitialReelColumns() {
  return Array.from({ length: 3 }, () => createIdleColumn())
}

function createInitialReelEngine(): ReelEngineState[] {
  return Array.from({ length: 3 }, () => ({
    phase: 'idle',
    position: getReelStartOffset(),
    decelStartAt: 0,
    decelStartPos: 0,
  }))
}

function createIdleColumn() {
  return createResultColumn(randomSymbol().id)
}

function createSpinningColumn() {
  return Array.from({ length: reelBufferCount + visibleSlotsPerReel + reelBufferCount }, () => randomSymbol())
}

function syncReelMetrics() {
  const symbolElement = reelFrameRef.value?.querySelector<HTMLElement>('.slot-machine__symbol')
  const stripElement = reelFrameRef.value?.querySelector<HTMLElement>('.slot-machine__strip')
  if (!symbolElement || !stripElement) return
  const nextSize = Math.round(symbolElement.getBoundingClientRect().height)
  const nextGap = Number.parseFloat(window.getComputedStyle(stripElement).rowGap || window.getComputedStyle(stripElement).gap || '0')
  if (nextSize > 0) {
    reelCellSize.value = nextSize
  }
  if (Number.isFinite(nextGap) && nextGap >= 0) {
    reelGap.value = nextGap
  }
}

function createResultColumn(centerSymbolId: string): ReelColumnState {
  const symbols = Array.from({ length: reelBufferCount }, () => randomSymbol())
  symbols.push(randomSymbol(), mapSymbol(centerSymbolId), randomSymbol())
  symbols.push(...Array.from({ length: reelBufferCount }, () => randomSymbol()))

  return {
    symbols,
    offsetY: getReelStartOffset(),
    landedAt: 0,
  }
}

function getReelStep() {
  return reelCellSize.value + reelGap.value
}

function getReelStartOffset() {
  return -(reelBufferCount * getReelStep())
}

function getReelThreshold() {
  return -((reelBufferCount - 1) * getReelStep())
}

function getReelTargetOffset() {
  return -(reelBufferCount * getReelStep())
}

function recycleColumnSymbol(columnIndex: number) {
  const nextColumns = [...reelColumns.value]
  const currentColumn = nextColumns[columnIndex]
  if (!currentColumn) return
  const symbols = [...currentColumn.symbols]
  const bottomSymbol = symbols.pop()
  if (!bottomSymbol) return
  symbols.unshift(randomSymbol())
  nextColumns[columnIndex] = {
    ...currentColumn,
    symbols,
  }
  reelColumns.value = nextColumns
}

function injectResultSymbols(columnIndex: number, symbolId: string) {
  const nextColumns = [...reelColumns.value]
  const currentColumn = nextColumns[columnIndex]
  if (!currentColumn) return
  const symbols = [...currentColumn.symbols]
  symbols[reelBufferCount] = randomSymbol()
  symbols[reelBufferCount + 1] = mapSymbol(symbolId)
  symbols[reelBufferCount + 2] = randomSymbol()
  nextColumns[columnIndex] = {
    ...currentColumn,
    symbols,
  }
  reelColumns.value = nextColumns
}

function tickReels(timestamp: number) {
  let allDone = true
  const elapsed = timestamp - reelStartTime.value
  const step = getReelStep()
  const threshold = getReelThreshold()
  const target = getReelTargetOffset()
  const resolvedSymbols = normalizeResultSymbols(pendingResultSymbols.value ?? [])

  const nextEngine: ReelEngineState[] = reelEngine.value.map((engine, columnIndex) => {
    if (engine.phase === 'done' || engine.phase === 'idle') {
      return engine
    }

    allDone = false

    if (engine.phase === 'spin') {
      const spinDuration = reelSpinDurations[columnIndex] ?? reelSpinDurations[reelSpinDurations.length - 1]
      if (elapsed >= spinDuration && pendingResultSymbols.value) {
        injectResultSymbols(columnIndex, resolvedSymbols[columnIndex] ?? randomSymbol().id)
        return {
          ...engine,
          phase: 'decel',
          decelStartAt: timestamp,
          decelStartPos: engine.position,
        }
      }

      let nextPosition = engine.position + reelSpinSpeedPx
      if (nextPosition >= threshold) {
        recycleColumnSymbol(columnIndex)
        nextPosition -= step
      }

      updateReelColumnOffset(columnIndex, nextPosition)

      return {
        ...engine,
        position: nextPosition,
      }
    }

    const decelElapsed = timestamp - engine.decelStartAt
    const progress = Math.min(decelElapsed / reelDecelDurationMs, 1)
    const eased = 1 - ((1 - progress) ** 2)
    const nextPosition = engine.decelStartPos + ((target - engine.decelStartPos) * eased)
    updateReelColumnOffset(columnIndex, nextPosition)

    if (progress >= 1) {
      markReelColumnLanded(columnIndex)
      return {
        ...engine,
        phase: 'done',
        position: target,
      }
    }

    return {
      ...engine,
      position: nextPosition,
    }
  })

  reelEngine.value = nextEngine

  if (allDone || nextEngine.every((engine) => engine.phase === 'done' || engine.phase === 'idle')) {
    clearReelAnimation()
    reelCompletionResolver?.()
    reelCompletionResolver = null
    return
  }

  reelAnimationFrame.value = window.requestAnimationFrame(tickReels)
}

function updateReelColumnOffset(columnIndex: number, offsetY: number) {
  const nextColumns = [...reelColumns.value]
  const currentColumn = nextColumns[columnIndex]
  if (!currentColumn) return
  nextColumns[columnIndex] = {
    ...currentColumn,
    offsetY,
  }
  reelColumns.value = nextColumns
}

function markReelColumnLanded(columnIndex: number) {
  const nextColumns = [...reelColumns.value]
  const currentColumn = nextColumns[columnIndex]
  if (!currentColumn) return
  nextColumns[columnIndex] = {
    ...currentColumn,
    offsetY: getReelTargetOffset(),
    landedAt: Date.now(),
  }
  reelColumns.value = nextColumns
}

async function waitForReelCompletion() {
  await (reelCompletionPromise ?? Promise.resolve())
  applyResultGrid(pendingResultSymbols.value ?? [])
}

function randomSymbol() {
  return symbolCatalog[Math.floor(Math.random() * symbolCatalog.length)] ?? fallbackSymbol
}

function mapSymbol(symbolId: string) {
  return symbolMap.get(symbolId) ?? fallbackSymbol
}

function recordSpin(resultState: GamePlayResult) {
  spinHistorySeed += 1
  spinHistory.value = [
    {
      id: spinHistorySeed,
      outcome: resultState.outcome,
      netAmount: resultState.net_amount,
      multiplier: resultState.multiplier,
      symbols: normalizeResultSymbols(resultState.symbols ?? []),
      createdAt: Date.now(),
    },
    ...spinHistory.value,
  ].slice(0, spinHistoryLimit)
}

function selectPresetBet(bet: number) {
  selectedBet.value = bet
  customBetInput.value = ''
}

function normalizeBet(currentBet: number, game: GameInfo | null | undefined) {
  const { minBet, maxBet } = getBetBounds(game)
  if (maxBet == null) {
    return Math.max(currentBet, minBet)
  }
  return Math.min(Math.max(currentBet, minBet), maxBet)
}

function normalizeSpinRounds(rounds: number) {
  const count = Math.floor(Number(rounds))
  if (!Number.isFinite(count)) return 1
  return Math.min(Math.max(count, 1), 10)
}

function getBetBounds(game: GameInfo | null | undefined) {
  const minBet = Math.max(1, Math.ceil(game?.min_bet ?? 10))
  const rawMaxBet = Number(game?.max_bet)
  const uiMaxBet = 100000000
  if (!Number.isFinite(rawMaxBet)) {
    return { minBet, maxBet: uiMaxBet }
  }

  const maxBet = Math.max(minBet, Math.max(Math.floor(rawMaxBet), uiMaxBet))
  return { minBet, maxBet }
}

function formatPlainAmount(value: number | null | undefined) {
  const amount = Number(value ?? 0)
  return formatNumber(Number.isFinite(amount) ? amount : 0)
}

function formatSignedAmount(value: number | null | undefined) {
  const amount = Number(value ?? 0)
  const safeAmount = Number.isFinite(amount) ? amount : 0
  if (safeAmount > 0) return `+${formatNumber(safeAmount)}`
  if (safeAmount < 0) return `${formatNumber(safeAmount)}`
  return '0.00'
}

function formatCompactBet(value: number) {
  if (value >= 10000) {
    return formatChineseAmountHint(value) || formatNumber(value)
  }
  return String(value)
}

function formatChineseAmountHint(value: number | null | undefined) {
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

function getAmountClass(value: number) {
  if (value > 0) return 'text-emerald-600 dark:text-emerald-400'
  if (value < 0) return 'text-red-600 dark:text-red-400'
  return 'text-gray-950 dark:text-white'
}

function historyNetClass(value: number) {
  return getAmountClass(value)
}

function formatHistoryTime(timestamp: number) {
  const date = new Date(timestamp)
  return `${pad2(date.getHours())}:${pad2(date.getMinutes())}:${pad2(date.getSeconds())}`
}

function pad2(value: number) {
  return String(value).padStart(2, '0')
}

function toErrorMessage(error: unknown) {
  return (error as { message?: string })?.message || '老虎机加载失败，请稍后再试。'
}
</script>

<style scoped>
.slots-page {
  --slots-symbol-size: clamp(68px, 9.4dvh, 82px);
  --slots-symbol-gap: clamp(8px, 1.2dvh, 10px);
  width: min(100%, 960px);
  min-height: calc(100dvh - 10rem);
  margin: 0 auto;
  color: #111827;
  display: grid;
  align-content: center;
  gap: 0.85rem;
}

.slots-page__header {
  width: 100%;
}

.slots-page__header-row {
  display: flex;
  align-items: flex-start;
  justify-content: flex-start;
}

.slots-page__heading {
  display: grid;
  justify-items: start;
  gap: 0.45rem;
}

.slots-page__title-block {
  display: grid;
  justify-items: start;
  gap: 0.18rem;
}

.slots-page__back {
  font-size: 0.8rem;
}

.slots-page__eyebrow {
  font-size: 0.68rem;
}

.slots-page__title {
  font-size: clamp(1.8rem, 3vw, 2.45rem);
  letter-spacing: 0;
}

.slots-title {
  display: inline-flex;
  align-items: center;
  gap: 0.65rem;
}

.slots-title__icon {
  display: inline-flex;
  width: 2.55rem;
  height: 2.55rem;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(229, 231, 235, 0.95);
  border-radius: 0.9rem;
  background: linear-gradient(180deg, #ffffff 0%, #f3f4f6 100%);
  font-size: 1.55rem;
  box-shadow:
    0 10px 22px rgba(15, 23, 42, 0.06),
    inset 0 1px 0 rgba(255, 255, 255, 0.98);
}

.dark .slots-title__icon {
  border-color: rgba(75, 85, 99, 0.82);
  background: linear-gradient(180deg, #1f2937 0%, #111827 100%);
}

.machine-panel,
.info-card {
  border: 1px solid rgba(229, 231, 235, 0.95);
  border-radius: 1.9rem;
  background: rgba(255, 255, 255, 0.96);
  box-shadow:
    0 24px 54px rgba(15, 23, 42, 0.05),
    inset 0 1px 0 rgba(255, 255, 255, 0.98);
}

.dark .machine-panel,
.dark .info-card {
  border-color: rgba(55, 65, 81, 0.82);
  background: rgba(17, 24, 39, 0.96);
  box-shadow:
    0 24px 54px rgba(0, 0, 0, 0.22),
    inset 0 1px 0 rgba(255, 255, 255, 0.03);
}

.machine-panel {
  padding: 1rem;
}

.machine-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  border: 1px solid rgba(235, 237, 240, 0.98);
  border-radius: 1.15rem;
  background: linear-gradient(180deg, #f9fafb 0%, #f3f4f6 100%);
  overflow: hidden;
}

.dark .machine-stats {
  border-color: rgba(55, 65, 81, 0.85);
  background: linear-gradient(180deg, #1f2937 0%, #111827 100%);
}

.machine-stats__item {
  padding: 0.7rem 0.8rem 0.66rem;
}

.machine-stats__item--centered {
  text-align: center;
}

.machine-stats__item + .machine-stats__item {
  border-left: 1px solid rgba(229, 231, 235, 0.9);
}

.dark .machine-stats__item + .machine-stats__item {
  border-left-color: rgba(55, 65, 81, 0.82);
}

.machine-stats__item span {
  display: block;
  font-size: 0.72rem;
  line-height: 1.2;
  color: #6b7280;
}

.dark .machine-stats__item span {
  color: #9ca3af;
}

.machine-stats__item strong {
  display: block;
  margin-top: 0.24rem;
  font-size: 0.92rem;
  font-weight: 700;
  line-height: 1.2;
  color: #111827;
}

.dark .machine-stats__item strong {
  color: #f9fafb;
}

.slot-machine {
  margin-top: 0.8rem;
}

.slot-machine__frame {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(235, 237, 240, 0.98);
  border-radius: 1.2rem;
  background: linear-gradient(180deg, #fafafa 0%, #f5f5f5 100%);
  padding: 0.54rem;
}

.dark .slot-machine__frame {
  border-color: rgba(55, 65, 81, 0.82);
  background: linear-gradient(180deg, #1f2937 0%, #111827 100%);
}

.slot-machine__frame--win {
  box-shadow:
    inset 0 0 0 1px rgba(16, 185, 129, 0.14),
    0 16px 34px rgba(16, 185, 129, 0.08);
}

.slot-machine__scanline {
  position: absolute;
  left: 0.65rem;
  right: 0.65rem;
  top: calc(50% + 4px);
  z-index: 2;
  height: 3px;
  transform: translateY(-50%);
  border-radius: 999px;
  background: linear-gradient(90deg, transparent, rgba(45, 212, 191, 0.75), transparent);
  box-shadow: 0 0 12px rgba(45, 212, 191, 0.2);
}

.slot-machine__reels {
  position: relative;
  z-index: 1;
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.48rem;
}

.slot-machine__reel {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(238, 239, 241, 0.98);
  border-radius: 0.98rem;
  background: linear-gradient(180deg, #fdfdfd 0%, #f6f7f8 100%);
  height: calc((var(--slots-symbol-size) * 3) + (var(--slots-symbol-gap) * 2) + 1rem);
  padding: 0.5rem;
  box-shadow:
    0 8px 18px rgba(15, 23, 42, 0.04),
    inset 0 1px 0 rgba(255, 255, 255, 0.96);
}

.dark .slot-machine__reel {
  border-color: rgba(75, 85, 99, 0.78);
  background: linear-gradient(180deg, #1f2937 0%, #0f172a 100%);
}

.slot-machine__reel--spinning {
  animation: slot-reel-glow 0.55s ease-in-out infinite alternate;
}

.slot-machine__strip {
  display: grid;
  gap: var(--slots-symbol-gap);
  will-change: transform;
}

.slot-machine__strip--spinning {
  backface-visibility: hidden;
}

.slot-machine__symbol {
  display: flex;
  height: var(--slots-symbol-size);
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(236, 238, 240, 0.98);
  border-radius: 0.78rem;
  background: linear-gradient(180deg, #ffffff 0%, #f7f7f8 100%);
  font-size: clamp(1.65rem, 2.8vw, 2.25rem);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.98);
}

.dark .slot-machine__symbol {
  border-color: rgba(75, 85, 99, 0.82);
  background: linear-gradient(180deg, #111827 0%, #0f172a 100%);
}

.slot-machine__symbol--center {
  border-color: rgba(110, 231, 183, 0.7);
  box-shadow:
    inset 0 0 0 1px rgba(167, 243, 208, 0.3),
    0 0 0 1px rgba(167, 243, 208, 0.16);
}

.slot-machine__symbol--winner {
  animation: slot-winner-pulse 0.72s ease-in-out infinite;
}

.slot-machine__symbol--landed {
  animation: slot-landed-bounce 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.machine-actions {
  margin-top: 0.65rem;
}

.machine-actions__group {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0.45rem;
}

.machine-actions__bet {
  border: 1px solid rgba(229, 231, 235, 0.96);
  border-radius: 0.78rem;
  background: linear-gradient(180deg, #ffffff 0%, #f7f7f8 100%);
  padding: 0.58rem 0.6rem;
  font-size: 0.92rem;
  font-weight: 700;
  color: #111827;
  transition: transform 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}

.machine-actions__bet:hover {
  transform: translateY(-1px);
}

.machine-actions__bet--active {
  border-color: rgba(244, 114, 182, 0.42);
  background: linear-gradient(180deg, #fff7fa 0%, #fdecef 100%);
  box-shadow: 0 10px 24px rgba(244, 114, 182, 0.08);
}

.dark .machine-actions__bet {
  border-color: rgba(75, 85, 99, 0.82);
  background: linear-gradient(180deg, #1f2937 0%, #111827 100%);
  color: #f9fafb;
}

.dark .machine-actions__bet--active {
  border-color: rgba(244, 114, 182, 0.45);
  background: linear-gradient(180deg, rgba(244, 114, 182, 0.12), rgba(17, 24, 39, 1));
}

.machine-actions__footer {
  margin-top: 0.6rem;
  display: grid;
  gap: 0.56rem;
}

.machine-actions__custom {
  display: grid;
  gap: 0.34rem;
}

.machine-actions__custom-input {
  position: relative;
}

.machine-actions__custom span {
  display: block;
  font-size: 0.7rem;
  font-weight: 600;
  color: #6b7280;
}

.dark .machine-actions__custom span {
  color: #9ca3af;
}

.machine-actions__custom input {
  width: 100%;
  border: 1px solid rgba(229, 231, 235, 0.96);
  border-radius: 0.82rem;
  background: linear-gradient(180deg, #ffffff 0%, #f7f7f8 100%);
  padding: 0.66rem 7.2rem 0.66rem 0.82rem;
  font-size: 0.92rem;
  font-weight: 600;
  color: #111827;
}

.machine-actions__custom-hint {
  position: absolute;
  top: 50%;
  right: 0.82rem;
  transform: translateY(-50%);
  font-size: 0.76rem;
  font-weight: 700;
  color: #6b7280;
  pointer-events: none;
}

.machine-actions__custom input:focus {
  outline: none;
  border-color: rgba(20, 184, 166, 0.45);
  box-shadow: 0 0 0 4px rgba(20, 184, 166, 0.1);
}

.machine-actions__custom input:disabled {
  cursor: not-allowed;
  opacity: 0.65;
}

.dark .machine-actions__custom input {
  border-color: rgba(75, 85, 99, 0.82);
  background: linear-gradient(180deg, #1f2937 0%, #111827 100%);
  color: #f9fafb;
}

.dark .machine-actions__custom-hint {
  color: #9ca3af;
}

.machine-actions__spin {
  width: 100%;
  border: 0;
  border-radius: 0.84rem;
  background: #151515;
  padding: 0.75rem 0.9rem;
  font-size: 0.98rem;
  font-weight: 700;
  letter-spacing: 0.01em;
  color: #ffffff;
  transition: transform 0.15s ease, opacity 0.15s ease, background 0.15s ease;
}

.machine-actions__spin:hover {
  background: #0f0f10;
}

.machine-actions__spin:active {
  transform: scale(0.99);
}

.machine-actions__spin:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.machine-actions__batch {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.48rem;
}

.machine-actions__batch button {
  border: 1px solid rgba(229, 231, 235, 0.96);
  border-radius: 0.78rem;
  background: linear-gradient(180deg, #ffffff 0%, #f7f7f8 100%);
  padding: 0.62rem 0.7rem;
  font-size: 0.88rem;
  font-weight: 700;
  color: #111827;
  transition: transform 0.15s ease, border-color 0.15s ease, box-shadow 0.15s ease;
}

.machine-actions__batch button:hover {
  transform: translateY(-1px);
}

.machine-actions__batch button:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.dark .machine-actions__batch button {
  border-color: rgba(75, 85, 99, 0.82);
  background: linear-gradient(180deg, #1f2937 0%, #111827 100%);
  color: #f9fafb;
}

.info-column {
  display: grid;
  gap: 0.7rem;
  align-content: start;
}

.info-card {
  padding: 0.82rem 0.9rem;
}

.info-card__titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.62rem;
  padding-bottom: 0.58rem;
  border-bottom: 1px solid rgba(235, 237, 240, 0.92);
}

.dark .info-card__titlebar {
  border-bottom-color: rgba(55, 65, 81, 0.82);
}

.info-card__titlebar--button {
  width: 100%;
  border: 0;
  background: transparent;
  padding-left: 0;
  padding-right: 0;
  text-align: left;
  cursor: pointer;
}

.info-card__titlebar--button:focus-visible {
  outline: 2px solid rgba(20, 184, 166, 0.45);
  outline-offset: 4px;
  border-radius: 0.75rem;
}

.info-card--history,
.info-card--odds {
  border-radius: 1.45rem;
}

.info-card--tip {
  border-radius: 1.35rem;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.98) 0%, rgba(250, 250, 249, 0.98) 100%);
}

.history-table {
  margin-top: 0.15rem;
}

.history-table__row {
  display: grid;
  grid-template-columns: 76px minmax(0, 1fr) 92px;
  align-items: center;
  gap: 0.5rem;
  padding: 0.56rem 0.05rem;
  border-bottom: 1px solid rgba(235, 237, 240, 0.92);
}

.dark .history-table__row {
  border-bottom-color: rgba(55, 65, 81, 0.82);
}

.history-table__time {
  font-size: 0.76rem;
  font-variant-numeric: tabular-nums;
  color: #6b7280;
}

.dark .history-table__time {
  color: #9ca3af;
}

.history-table__symbols {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  min-width: 0;
}

.history-table__symbol {
  font-size: 0.86rem;
}

.history-table__amount {
  text-align: right;
  font-size: 0.84rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.history-table__empty {
  padding: 0.7rem 0 0.05rem;
  font-size: 0.82rem;
  color: #6b7280;
}

.dark .history-table__empty {
  color: #9ca3af;
}

.odds-table {
  margin-top: 0.2rem;
}

.odds-toggle {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  flex: 0 0 auto;
  font-size: 0.78rem;
  font-weight: 700;
  color: #6b7280;
}

.dark .odds-toggle {
  color: #9ca3af;
}

.odds-toggle__chevron {
  display: inline-block;
  transition: transform 0.18s ease;
}

.odds-toggle__chevron--open {
  transform: rotate(180deg);
}

.odds-table__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 0.6rem;
  padding: 0.6rem 0.05rem;
  border-bottom: 1px solid rgba(235, 237, 240, 0.92);
}

.odds-table__row:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.dark .odds-table__row {
  border-bottom-color: rgba(55, 65, 81, 0.82);
}

.odds-table__pattern {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.86rem;
  color: #111827;
}

.dark .odds-table__pattern {
  color: #f9fafb;
}

.odds-table__label {
  font-size: 0.76rem;
  color: #6b7280;
}

.dark .odds-table__label {
  color: #9ca3af;
}

.odds-table__value {
  font-size: 0.86rem;
  font-weight: 700;
  color: #111827;
}

.dark .odds-table__value {
  color: #f9fafb;
}

@keyframes slot-reel-glow {
  from {
    box-shadow: 0 8px 18px rgba(15, 23, 42, 0.04), inset 0 0 0 1px rgba(20, 184, 166, 0.08);
  }

  to {
    box-shadow: 0 12px 24px rgba(15, 23, 42, 0.06), inset 0 0 0 1px rgba(20, 184, 166, 0.14);
  }
}

@keyframes slot-winner-pulse {
  0%,
  100% {
    transform: scale(1);
  }

  50% {
    transform: scale(1.04);
  }
}

@keyframes slot-landed-bounce {
  0% {
    transform: scale(0.3);
    opacity: 0;
  }

  50% {
    transform: scale(1.18);
  }

  100% {
    transform: scale(1);
    opacity: 1;
  }
}

@media (max-width: 1279px) {
  .history-table__row {
    grid-template-columns: 72px minmax(0, 1fr) 88px;
  }
}

@media (min-width: 960px) {
  .slots-page__header {
    width: 832px;
    justify-self: center;
  }

  .slots-page__header-row {
    width: 430px;
  }

  .slots-layout {
    display: grid;
    grid-template-columns: minmax(0, 430px) minmax(0, 390px);
    align-items: start;
    justify-content: center;
    gap: 0.9rem;
  }

  .machine-panel {
    width: 100%;
    max-width: 430px;
    min-width: 0;
    justify-self: start;
  }

  .info-column {
    width: 390px;
    min-width: 0;
    align-self: start;
  }
}

@media (min-width: 960px) and (max-width: 1279px) {
  .slots-page {
    width: min(100%, 520px);
  }

  .slots-page__header {
    width: 430px;
    justify-self: center;
  }

  .slots-page__header-row {
    width: 100%;
  }

  .slots-layout {
    grid-template-columns: minmax(0, 430px);
  }

  .machine-panel {
    justify-self: center;
  }

  .info-column {
    display: none;
  }
}

@media (min-width: 960px) and (max-height: 920px) {
  .slots-page {
    --slots-symbol-size: clamp(54px, 7.6dvh, 62px);
    --slots-symbol-gap: 6px;
    gap: 0.55rem;
  }

  .slots-page__heading {
    gap: 0.25rem;
  }

  .slots-page__eyebrow {
    display: none;
  }

  .slots-page__title {
    font-size: clamp(1.5rem, 2.4vw, 1.95rem);
  }

  .slots-title__icon {
    width: 2.1rem;
    height: 2.1rem;
    font-size: 1.25rem;
  }

  .machine-panel {
    padding: 0.72rem;
  }

  .machine-stats__item {
    padding: 0.46rem 0.65rem 0.42rem;
  }

  .machine-stats__item span {
    font-size: 0.68rem;
  }

  .machine-stats__item strong {
    margin-top: 0.18rem;
    font-size: 0.86rem;
  }

  .slot-machine {
    margin-top: 0.45rem;
  }

  .slot-machine__frame {
    padding: 0.46rem;
  }

  .slot-machine__reel {
    height: calc((var(--slots-symbol-size) * 3) + (var(--slots-symbol-gap) * 2) + 0.82rem);
    padding: 0.41rem;
  }

  .slot-machine__symbol {
    font-size: clamp(1.45rem, 2.35vw, 1.9rem);
  }

  .machine-actions {
    margin-top: 0.42rem;
  }

  .machine-actions__group {
    gap: 0.38rem;
  }

  .machine-actions__bet {
    padding: 0.45rem 0.55rem;
    font-size: 0.84rem;
  }

  .machine-actions__footer {
    margin-top: 0.42rem;
    gap: 0.42rem;
  }

  .machine-actions__custom {
    gap: 0.24rem;
  }

  .machine-actions__custom span {
    font-size: 0.66rem;
  }

  .machine-actions__custom input {
    padding: 0.5rem 0.7rem;
    font-size: 0.84rem;
  }

  .machine-actions__custom-hint {
    right: 0.7rem;
    font-size: 0.7rem;
  }

  .machine-actions__spin {
    padding: 0.58rem 0.8rem;
    font-size: 0.9rem;
  }

  .machine-actions__batch {
    gap: 0.38rem;
  }

  .machine-actions__batch button {
    padding: 0.48rem 0.6rem;
    font-size: 0.8rem;
  }

  .info-card--tip {
    display: none;
  }
}

@media (min-width: 960px) and (max-height: 740px) {
  .slots-page {
    --slots-symbol-size: clamp(48px, 7dvh, 54px);
    --slots-symbol-gap: 5px;
    min-height: calc(100dvh - 9rem);
  }

  .slots-page__back {
    display: none;
  }

  .slots-page__title {
    font-size: clamp(1.32rem, 2.1vw, 1.7rem);
  }

  .slots-title__icon {
    width: 1.86rem;
    height: 1.86rem;
    font-size: 1.08rem;
  }

  .slots-layout {
    gap: 0.6rem;
  }
}

@media (max-width: 767px) {
  .slots-page {
    min-height: auto;
    align-content: start;
    gap: 0.9rem;
  }

  .machine-panel {
    padding: 1.1rem;
  }

  .slots-title {
    gap: 0.7rem;
  }

  .slots-title__icon {
    width: 2.7rem;
    height: 2.7rem;
    font-size: 1.7rem;
  }

  .machine-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .slot-machine__reels {
    gap: 0.55rem;
  }

  .slot-machine__reel {
    padding: 0.55rem;
  }

  .slot-machine__symbol {
    font-size: clamp(1.75rem, 8vw, 2.4rem);
  }

  .machine-actions__group {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .machine-actions__batch {
    grid-template-columns: 1fr;
  }

  .history-table__row {
    grid-template-columns: 1fr;
    gap: 0.45rem;
  }

  .history-table__amount {
    text-align: left;
  }
}
</style>
