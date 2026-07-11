import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
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
    try {
      lastExchange.value = await exchangeGameBalance(direction, amount)
      await refresh()
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
    try {
      lastRound.value = await playGame(gameType, betAmount)
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
