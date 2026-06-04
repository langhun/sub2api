import { apiClient } from './client'

export type GameType = 'slots' | 'train' | 'texas'
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
  balance: number
  games: GameInfo[]
}

export interface GamePlayResult {
  game_type: GameType
  bet_amount: number
  payout_amount: number
  net_amount: number
  multiplier: number
  balance_before: number
  balance_after: number
  outcome: GameOutcome
  symbols?: string[]
  message: string
}

function buildGameIdempotencyKey(gameType: GameType, betAmount: number): string {
  const randomPart =
    typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `game:${gameType}:${betAmount}:${randomPart}`
}

export const gamesAPI = {
  async getHall(): Promise<GameHallStatus> {
    const { data } = await apiClient.get<GameHallStatus>('/games/hall')
    return data
  },

  async play(gameType: GameType, betAmount: number): Promise<GamePlayResult> {
    const { data } = await apiClient.post<GamePlayResult>('/games/play', {
      game_type: gameType,
      bet_amount: betAmount,
    }, {
      headers: {
        'Idempotency-Key': buildGameIdempotencyKey(gameType, betAmount),
      },
    })
    return data
  },
}

export default gamesAPI
