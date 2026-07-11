import { apiClient } from './client'

export interface GameHallGame {
  type: string
  name: string
  description?: string
  min_bet: number
  max_bet: number
  multipliers: number[]
}

export interface GameHallStatus {
  main_balance: number
  dg_balance: number
  jackpot_balance: number
  games: GameHallGame[]
}

export interface GameExchangeResult {
  direction: 'balance_to_dg' | 'dg_to_balance'
  amount: number
  main_balance_before: number
  main_balance_after: number
  dg_balance_before: number
  dg_balance_after: number
}

export interface GamePlayResult {
	round_id?: number
  game_type: string
  bet_amount: number
  payout_amount: number
  net_amount: number
  multiplier: number
  dg_balance_before: number
  dg_balance_after: number
  jackpot_balance: number
  outcome: 'win' | 'loss' | 'push' | string
  symbols: string[]
  message: string
}

export interface GameWalletTransaction {
  id: number
  tx_type: string
  amount: number
  balance_before: number
  balance_after: number
  created_at: string
}

export interface GameRound {
  id: number
  game_type: string
  bet_amount: number
  payout_amount: number
  net_amount: number
  multiplier: number
  outcome: string
  symbols: string[]
  created_at: string
}

export interface GameHallPage<T> { items: T[]; total: number; page: number; page_size: number }

export function createGameHallIdempotencyKey(scope: string): string {
  const suffix = typeof crypto !== 'undefined' && crypto.randomUUID
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `${scope}-${suffix}`
}

export async function getGameHallStatus(): Promise<GameHallStatus> {
  const { data } = await apiClient.get<GameHallStatus>('/user/game-hall/status')
  return data
}

export async function exchangeGameBalance(
  direction: GameExchangeResult['direction'],
  amount: number,
  requestKey?: string,
): Promise<GameExchangeResult> {
  const { data } = await apiClient.post<GameExchangeResult>(
    '/user/game-hall/exchange',
    { direction, amount },
    { headers: { 'Idempotency-Key': requestKey || createGameHallIdempotencyKey('game-exchange') } },
  )
  return data
}

export async function playGame(gameType: string, betAmount: number, requestKey?: string): Promise<GamePlayResult> {
  const { data } = await apiClient.post<GamePlayResult>(
    '/user/game-hall/play',
    { game_type: gameType, bet_amount: betAmount },
    { headers: { 'Idempotency-Key': requestKey || createGameHallIdempotencyKey('game-play') } },
  )
  return data
}

export async function getGameTransactions(page = 1, pageSize = 10): Promise<GameHallPage<GameWalletTransaction>> {
  const { data } = await apiClient.get<GameHallPage<GameWalletTransaction>>('/user/game-hall/transactions', { params: { page, page_size: pageSize } })
  return data
}

export async function getGameRounds(page = 1, pageSize = 10): Promise<GameHallPage<GameRound>> {
  const { data } = await apiClient.get<GameHallPage<GameRound>>('/user/game-hall/rounds', { params: { page, page_size: pageSize } })
  return data
}

export const gameHallAPI = { getGameHallStatus, exchangeGameBalance, playGame, getGameTransactions, getGameRounds }
