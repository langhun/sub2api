<template>
  <AppLayout>
    <div class="game-hall mx-auto max-w-6xl space-y-5">
      <header class="flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <p class="text-sm font-medium text-[var(--muted-foreground)]">游戏中心</p>
          <h1 class="text-2xl font-semibold text-[var(--foreground)]">娱乐大厅</h1>
        </div>
        <button class="btn btn-secondary" type="button" :disabled="loading" @click="loadHall">
          {{ loading ? '刷新中' : '刷新余额' }}
        </button>
      </header>

      <section class="hall-summary">
        <span>娱乐余额</span>
        <strong>{{ loading ? '加载中' : formatMoney(gameHall?.balance) }}</strong>
      </section>

      <p v-if="error" class="rounded-md border border-[var(--destructive)]/30 bg-[var(--destructive)]/10 px-4 py-3 text-sm text-[var(--destructive)]">
        {{ error }}
      </p>

      <main class="grid gap-4 md:grid-cols-2">
        <RouterLink class="game-card" data-testid="slots-entry" to="/games/slots">
          <span class="game-mark" aria-hidden="true">7</span>
          <span class="game-content">
            <span class="game-title">老虎机</span>
            <span class="game-desc">三轴滚动，拉杆后独立结算。</span>
            <span class="game-meta">{{ slotMetaText }}</span>
          </span>
          <span class="game-action" aria-hidden="true">进入</span>
        </RouterLink>

        <RouterLink class="game-card" data-testid="ssq-entry" to="/games/ssq">
          <span class="game-mark ball-mark" aria-hidden="true">双</span>
          <span class="game-content">
            <span class="game-title">双色球</span>
            <span class="game-desc">选择 6 个红球和 1 个蓝球。</span>
            <span class="game-meta">独立选号页</span>
          </span>
          <span class="game-action" aria-hidden="true">进入</span>
        </RouterLink>
      </main>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { gamesAPI, type GameHallStatus } from '@/api/games'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAppStore } from '@/stores'
import { formatDualDisplayAmount } from '@/utils/format'

const appStore = useAppStore()

const gameHall = ref<GameHallStatus | null>(null)
const loading = ref(false)
const error = ref('')

const slotGame = computed(() => gameHall.value?.games.find((game) => game.type === 'slots') ?? null)
const slotMetaText = computed(() => {
  if (loading.value) return '正在读取状态'
  if (slotGame.value) return `单次 ${formatMoney(slotGame.value.min_bet)} - ${formatMoney(slotGame.value.max_bet)}`
  if (error.value) return '入口可用，状态稍后刷新'
  return '等待状态加载'
})

onMounted(() => {
  void loadHall()
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

function formatMoney(value: string | number | null | undefined) {
  const amount = Number(value ?? 0)
  return `${formatDualDisplayAmount(Number.isFinite(amount) ? amount : 0).display} DG`
}

function readableError(value: unknown) {
  return (value as { message?: string })?.message || '娱乐大厅状态加载失败，请稍后重试。'
}
</script>

<style scoped>
.game-hall {
  padding-bottom: 2rem;
}

.hall-summary,
.game-card {
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--card);
  box-shadow: 0 1px 2px rgb(15 23 42 / 0.05);
}

.hall-summary {
  min-height: 5.25rem;
  padding: 1rem;
}

.hall-summary span {
  display: block;
  font-size: 0.875rem;
  color: var(--muted-foreground);
}

.hall-summary strong {
  display: block;
  margin-top: 0.5rem;
  overflow-wrap: anywhere;
  font-size: 1.25rem;
  color: var(--foreground);
}

.game-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 1rem;
  align-items: center;
  min-height: 9rem;
  padding: 1.25rem;
  color: var(--foreground);
  text-decoration: none;
  transition: border-color 0.15s ease, transform 0.15s ease, box-shadow 0.15s ease;
}

.game-card:hover {
  border-color: var(--primary);
  box-shadow: 0 10px 24px rgb(15 23 42 / 0.1);
  transform: translateY(-1px);
}

.game-mark {
  display: inline-flex;
  width: 3.25rem;
  height: 3.25rem;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  background: rgb(239 68 68 / 0.12);
  color: rgb(185 28 28);
  font-size: 1.75rem;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
}

.ball-mark {
  background: rgb(37 99 235 / 0.12);
  color: rgb(29 78 216);
  font-size: 1.25rem;
}

.game-content {
  display: grid;
  gap: 0.35rem;
  min-width: 0;
}

.game-title {
  font-size: 1.125rem;
  font-weight: 700;
}

.game-desc,
.game-meta {
  overflow-wrap: anywhere;
  color: var(--muted-foreground);
  font-size: 0.875rem;
}

.game-action {
  border-radius: 6px;
  background: var(--primary);
  color: var(--primary-foreground);
  padding: 0.45rem 0.75rem;
  font-size: 0.875rem;
  font-weight: 700;
}

@media (max-width: 640px) {
  .game-card {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .game-action {
    grid-column: 1 / -1;
    justify-self: start;
  }
}
</style>
