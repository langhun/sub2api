import { apiClient } from './client'

export type GameType = 'slots'
export type GameExchangeDirection = 'balance_to_dg' | 'dg_to_balance'
export type GameOutcome = 'win' | 'lose' | 'draw'

export interface GameInfo {
  type: GameType
  name: string
  description: string
  min_bet: number
  max_bet: number
  multipliers: number[]
}

export interface GameHallStatus {
  main_balance: number
  dg_balance: number
  jackpot_balance: number
  games: GameInfo[]
}

export interface GameExchangeResult {
  direction: GameExchangeDirection
  amount: number
  main_balance_before: number
  main_balance_after: number
  dg_balance_before: number
  dg_balance_after: number
}

export interface GamePlayResult {
  game_type: GameType
  bet_amount: number
  payout_amount: number
  net_amount: number
  multiplier: number
  dg_balance_before: number
  dg_balance_after: number
  jackpot_balance: number
  outcome: GameOutcome
  symbols?: string[]
  message: string
}

function createGameIdempotencyKey(prefix: string, amount: number) {
  const random =
    typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `${prefix}:${amount}:${random}`
}

export const gamesAPI = {
  async getHall(): Promise<GameHallStatus> {
    const { data } = await apiClient.get<GameHallStatus>('/games/hall')
    return data
  },

  async exchange(
    direction: GameExchangeDirection,
    amount: number,
    idempotencyKey = createGameIdempotencyKey(`exchange:${direction}`, amount),
  ): Promise<GameExchangeResult> {
    const { data } = await apiClient.post<GameExchangeResult>(
      '/games/exchange',
      {
        direction,
        amount,
      },
      {
        headers: {
          'Idempotency-Key': idempotencyKey,
        },
      },
    )
    return data
  },

  async play(
    gameType: GameType,
    betAmount: number,
    idempotencyKey = createGameIdempotencyKey(`game:${gameType}`, betAmount),
  ): Promise<GamePlayResult> {
    const { data } = await apiClient.post<GamePlayResult>(
      '/games/play',
      {
        game_type: gameType,
        bet_amount: betAmount,
      },
      {
        headers: {
          'Idempotency-Key': idempotencyKey,
        },
      },
    )
    return data
  },
}

export default gamesAPI
