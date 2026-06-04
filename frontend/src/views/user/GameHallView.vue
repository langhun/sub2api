<template>
  <AppLayout>
    <div class="lottery-page mx-auto max-w-7xl space-y-5">
      <header class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <p class="text-sm font-medium text-[var(--muted-foreground)]">娱乐大厅</p>
          <h1 class="text-2xl font-semibold text-[var(--foreground)]">双色球</h1>
        </div>
        <button class="btn btn-secondary" type="button" :disabled="loadingCurrent || ordersLoading" @click="refreshAll">
          刷新
        </button>
      </header>

      <section class="grid gap-3 md:grid-cols-4">
        <div class="summary-cell">
          <span>当前期号</span>
          <strong>{{ current?.issue_no || '-' }}</strong>
        </div>
        <div class="summary-cell">
          <span>截止时间</span>
          <strong>{{ formatDateTime(current?.cutoff_time) || '-' }}</strong>
        </div>
        <div class="summary-cell">
          <span>当前奖池</span>
          <strong :title="formatMoneyTitle(current?.jackpot_balance)">{{ formatMoney(current?.jackpot_balance) }}</strong>
        </div>
        <div class="summary-cell">
          <span>投注状态</span>
          <strong :class="current?.is_closed ? 'text-[var(--destructive)]' : 'text-emerald-600 dark:text-emerald-300'">
            {{ currentStatusText }}
          </strong>
        </div>
      </section>

      <p v-if="currentError" class="rounded-md border border-[var(--destructive)]/30 bg-[var(--destructive)]/10 px-4 py-3 text-sm text-[var(--destructive)]">
        {{ currentError }}
      </p>

      <main class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_24rem]">
        <section class="lottery-panel">
          <div class="flex flex-col gap-2 border-b border-[var(--border)] pb-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 class="text-lg font-semibold text-[var(--foreground)]">选号区</h2>
              <p class="text-sm text-[var(--muted-foreground)]">单注 100 DG，选择 6 个红球和 1 个蓝球。</p>
            </div>
            <div class="text-sm text-[var(--muted-foreground)]">已选 {{ selectedRedBalls.length }}/6 + {{ selectedBlueBall ? 1 : 0 }}/1</div>
          </div>

          <div class="space-y-5 pt-4">
            <div>
              <div class="mb-3 flex items-center justify-between gap-3">
                <h3 class="text-sm font-semibold text-[var(--foreground)]">红球</h3>
                <span class="text-xs text-[var(--muted-foreground)]">01-33</span>
              </div>
              <div class="ball-grid red-grid" data-testid="red-ball-grid">
                <button
                  v-for="ball in redBallOptions"
                  :key="ball"
                  type="button"
                  class="ball-button red-ball"
                  :class="{ selected: selectedRedSet.has(ball) }"
                  :aria-pressed="selectedRedSet.has(ball)"
                  :aria-label="`红球 ${padBall(ball)}`"
                  :disabled="submitting"
                  :data-testid="`red-ball-${ball}`"
                  @click="toggleRedBall(ball)"
                >
                  {{ padBall(ball) }}
                </button>
              </div>
            </div>

            <div>
              <div class="mb-3 flex items-center justify-between gap-3">
                <h3 class="text-sm font-semibold text-[var(--foreground)]">蓝球</h3>
                <span class="text-xs text-[var(--muted-foreground)]">01-16</span>
              </div>
              <div class="ball-grid blue-grid" data-testid="blue-ball-grid">
                <button
                  v-for="ball in blueBallOptions"
                  :key="ball"
                  type="button"
                  class="ball-button blue-ball"
                  :class="{ selected: selectedBlueBall === ball }"
                  :aria-pressed="selectedBlueBall === ball"
                  :aria-label="`蓝球 ${padBall(ball)}`"
                  :disabled="submitting"
                  :data-testid="`blue-ball-${ball}`"
                  @click="selectBlueBall(ball)"
                >
                  {{ padBall(ball) }}
                </button>
              </div>
            </div>
          </div>

          <div class="mt-5 flex flex-col gap-3 border-t border-[var(--border)] pt-4 sm:flex-row sm:items-center sm:justify-between">
            <div class="min-h-5 text-sm text-[var(--muted-foreground)]">{{ submitHint }}</div>
            <div class="grid grid-cols-3 gap-2 sm:flex sm:justify-end">
              <button class="btn btn-secondary" type="button" :disabled="submitting" @click="randomPick">机选一注</button>
              <button class="btn btn-secondary" type="button" :disabled="submitting || nothingSelected" @click="clearSelection">清空</button>
              <button class="btn btn-primary" type="button" :disabled="!canSubmit" data-testid="submit-bet" @click="submitBet">
                {{ submitting ? '提交中' : '立即投注' }}
              </button>
            </div>
          </div>
        </section>

        <aside class="lottery-panel">
          <h2 class="text-lg font-semibold text-[var(--foreground)]">我的投注</h2>
          <p class="mt-1 text-sm text-[var(--muted-foreground)]">显示当前期号的最近投注记录。</p>

          <div class="mt-4 space-y-3">
            <div v-if="ordersLoading" class="order-empty">加载中</div>
            <div v-else-if="orders.length === 0" class="order-empty">暂无投注记录</div>
            <article v-for="order in orders" v-else :key="order.order_id" class="order-row">
              <div class="flex items-center justify-between gap-3">
                <strong class="text-sm text-[var(--foreground)]">第 {{ order.issue_no }} 期</strong>
                <span class="rounded-md bg-[var(--muted)] px-2 py-1 text-xs text-[var(--muted-foreground)]">{{ order.status }}</span>
              </div>
              <div class="mt-2 flex flex-wrap gap-1.5">
                <span v-for="ball in order.red_balls" :key="`${order.order_id}-r-${ball}`" class="mini-ball mini-red">{{ ball }}</span>
                <span class="mini-ball mini-blue">{{ order.blue_ball }}</span>
              </div>
              <div class="mt-2 grid gap-1 text-xs text-[var(--muted-foreground)]">
                <span>金额：{{ formatMoney(order.cost) }}</span>
                <span>奖励：{{ formatMoney(order.reward) }}</span>
                <span>{{ formatDateTime(order.created_at) }}</span>
              </div>
            </article>
          </div>
        </aside>
      </main>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { lotteryAPI, type LotteryCurrent, type LotteryOrder } from '@/api/lottery'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores'
import { formatDateTime, formatDualDisplayAmount } from '@/utils/format'

const appStore = useAppStore()
const redBallOptions = range(1, 33)
const blueBallOptions = range(1, 16)

const current = ref<LotteryCurrent | null>(null)
const orders = ref<LotteryOrder[]>([])
const selectedRedBalls = ref<number[]>([])
const selectedBlueBall = ref<number | null>(null)
const loadingCurrent = ref(false)
const ordersLoading = ref(false)
const submitting = ref(false)
const currentError = ref('')

const selectedRedSet = computed(() => new Set(selectedRedBalls.value))
const nothingSelected = computed(() => selectedRedBalls.value.length === 0 && selectedBlueBall.value === null)
const currentStatusText = computed(() => {
  if (loadingCurrent.value) return '加载中'
  if (!current.value) return '未加载'
  return current.value.is_closed ? '已截止' : '可投注'
})
const canSubmit = computed(() =>
  Boolean(current.value) &&
  !current.value?.is_closed &&
  selectedRedBalls.value.length === 6 &&
  selectedBlueBall.value !== null &&
  !submitting.value
)
const submitHint = computed(() => {
  if (!current.value) return '请先等待当前期号加载完成。'
  if (current.value.is_closed) return '本期已截止投注。'
  if (selectedRedBalls.value.length < 6) return `还需要选择 ${6 - selectedRedBalls.value.length} 个红球。`
  if (selectedBlueBall.value === null) return '请选择 1 个蓝球。'
  return '号码已就绪，可以投注。'
})

onMounted(() => {
  void refreshAll()
})

async function refreshAll() {
  await loadCurrent()
  await loadOrders()
}

async function loadCurrent() {
  loadingCurrent.value = true
  currentError.value = ''
  try {
    current.value = await lotteryAPI.getCurrent()
  } catch (error) {
    current.value = null
    currentError.value = readableError(error)
    appStore.showError(currentError.value)
  } finally {
    loadingCurrent.value = false
  }
}

async function loadOrders() {
  ordersLoading.value = true
  try {
    orders.value = await lotteryAPI.getOrders(current.value?.issue_no)
  } catch (error) {
    orders.value = []
    appStore.showError(readableError(error))
  } finally {
    ordersLoading.value = false
  }
}

function toggleRedBall(ball: number) {
  if (selectedRedSet.value.has(ball)) {
    selectedRedBalls.value = selectedRedBalls.value.filter((item) => item !== ball)
    return
  }
  if (selectedRedBalls.value.length >= 6) return
  selectedRedBalls.value = [...selectedRedBalls.value, ball].sort((a, b) => a - b)
}

function selectBlueBall(ball: number) {
  selectedBlueBall.value = selectedBlueBall.value === ball ? null : ball
}

function randomPick() {
  selectedRedBalls.value = shuffle(redBallOptions).slice(0, 6).sort((a, b) => a - b)
  selectedBlueBall.value = blueBallOptions[Math.floor(Math.random() * blueBallOptions.length)]
}

function clearSelection() {
  selectedRedBalls.value = []
  selectedBlueBall.value = null
}

async function submitBet() {
  if (!canSubmit.value || selectedBlueBall.value === null || !current.value) return
  submitting.value = true
  try {
    await lotteryAPI.createBet({
      red_balls: selectedRedBalls.value,
      blue_ball: selectedBlueBall.value,
    }, current.value.issue_no)
    appStore.showSuccess('投注成功，已刷新投注记录')
    clearSelection()
    await refreshAll()
  } catch (error) {
    appStore.showError(readableError(error))
  } finally {
    submitting.value = false
  }
}

function range(start: number, end: number) {
  return Array.from({ length: end - start + 1 }, (_, index) => start + index)
}

function shuffle(values: number[]) {
  return [...values].sort(() => Math.random() - 0.5)
}

function padBall(value: number | string) {
  return String(value).padStart(2, '0')
}

function formatMoney(value: string | number | null | undefined) {
  const amount = Number(value ?? 0)
  return `${formatDualDisplayAmount(Number.isFinite(amount) ? amount : 0).display} DG`
}

function formatMoneyTitle(value: string | number | null | undefined) {
  const amount = Number(value ?? 0)
  return `${formatDualDisplayAmount(Number.isFinite(amount) ? amount : 0).full} DG`
}

function readableError(error: unknown) {
  const value = error as { message?: string; reason?: string; metadata?: Record<string, string> }
  if (value?.reason === 'LOTTERY_ISSUE_CLOSED') return '本期已截止投注，请等待下一期。'
  if (value?.reason === 'LOTTERY_NUMBERS_INVALID') return '号码格式不正确，请检查红球和蓝球。'
  if (value?.reason === 'LOTTERY_BET_LIMIT_EXCEEDED') return '本期投注数量已达到上限。'
  if (value?.reason === 'BANK_INSUFFICIENT_BALANCE') return 'DG 币余额不足，无法投注。'
  return value?.message || '请求失败，请稍后重试。'
}
</script>

<style scoped>
.lottery-page {
  padding-bottom: 2rem;
}

.summary-cell,
.lottery-panel {
  border: 1px solid var(--border);
  background: var(--card);
  border-radius: 8px;
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

.lottery-panel {
  padding: 1.25rem;
}

.ball-grid {
  display: grid;
  gap: 0.5rem;
}

.red-grid {
  grid-template-columns: repeat(auto-fill, minmax(2.75rem, 1fr));
}

.blue-grid {
  grid-template-columns: repeat(auto-fill, minmax(2.75rem, 1fr));
}

.ball-button {
  aspect-ratio: 1;
  min-width: 2.75rem;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--background);
  color: var(--foreground);
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  transition: background-color 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.red-ball.selected {
  border-color: rgb(220 38 38);
  background: rgb(220 38 38);
  color: white;
}

.blue-ball.selected {
  border-color: rgb(37 99 235);
  background: rgb(37 99 235);
  color: white;
}

.order-row {
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 0.875rem;
}

.order-empty {
  border: 1px dashed var(--border);
  border-radius: 8px;
  padding: 2rem 1rem;
  text-align: center;
  color: var(--muted-foreground);
}

.mini-ball {
  display: inline-flex;
  width: 1.75rem;
  height: 1.75rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: white;
  font-size: 0.75rem;
  font-variant-numeric: tabular-nums;
  font-weight: 700;
}

.mini-red {
  background: rgb(220 38 38);
}

.mini-blue {
  background: rgb(37 99 235);
}
</style>
