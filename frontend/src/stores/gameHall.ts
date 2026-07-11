import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  createGameHallIdempotencyKey,
  exchangeGameBalance,
  getGameHallStatus,
  playGame,
  type GameExchangeResult,
  type GameHallStatus,
  type GamePlayResult,
} from '@/api/gameHall'

export const useGameHallStore = defineStore('gameHall', () => {
  const status = ref<GameHallStatus | null>(null)
  const lastExchange = ref<GameExchangeResult | null>(null)
  const lastRound = ref<GamePlayResult | null>(null)
  const loading = ref(false)
  const submitting = ref(false)
  const error = ref('')
  const exchangeAttempt = ref<{ signature: string; key: string } | null>(null)
  const playAttempt = ref<{ signature: string; key: string } | null>(null)

  const enabledGames = computed(() => status.value?.games ?? [])

  async function refresh(): Promise<void> {
    loading.value = true
    error.value = ''
    try {
      status.value = await getGameHallStatus()
    } catch (cause) {
      error.value = getErrorMessage(cause)
      throw cause
    } finally {
      loading.value = false
    }
  }

  async function exchange(direction: GameExchangeResult['direction'], amount: number): Promise<void> {
    submitting.value = true
    error.value = ''
    const signature = JSON.stringify([direction, amount])
    if (exchangeAttempt.value?.signature !== signature) {
      exchangeAttempt.value = { signature, key: createGameHallIdempotencyKey('game-exchange') }
    }
    try {
      lastExchange.value = await exchangeGameBalance(direction, amount, exchangeAttempt.value.key)
      exchangeAttempt.value = null
      if (status.value) {
        status.value.main_balance = lastExchange.value.main_balance_after
        status.value.dg_balance = lastExchange.value.dg_balance_after
      }
    } catch (cause) {
      error.value = getErrorMessage(cause)
      throw cause
    } finally {
      submitting.value = false
    }
  }

  async function play(gameType: string, betAmount: number): Promise<void> {
    submitting.value = true
    error.value = ''
    const signature = JSON.stringify([gameType, betAmount])
    if (playAttempt.value?.signature !== signature) {
      playAttempt.value = { signature, key: createGameHallIdempotencyKey('game-play') }
    }
    try {
      lastRound.value = await playGame(gameType, betAmount, playAttempt.value.key)
      playAttempt.value = null
      if (status.value) {
        status.value.dg_balance = lastRound.value.dg_balance_after
        status.value.jackpot_balance = lastRound.value.jackpot_balance
      }
    } catch (cause) {
      error.value = getErrorMessage(cause)
      throw cause
    } finally {
      submitting.value = false
    }
  }

  function clearResult(): void {
    lastExchange.value = null
    lastRound.value = null
  }

  return { status, lastExchange, lastRound, loading, submitting, error, enabledGames, refresh, exchange, play, clearResult }
})

function getErrorMessage(cause: unknown): string {
  const responseMessage = (cause as { response?: { data?: { detail?: string; error?: string; message?: string } } })
    ?.response?.data
  return responseMessage?.detail || responseMessage?.error || responseMessage?.message || (cause as Error)?.message || 'Request failed'
}
