import { apiClient } from './client'

export interface LeaderboardEntry {
  rank: number
  username: string
  value: number
  extra_int?: number
  extra_int2?: number
  extra_float?: number
  extra_date?: string
}

export interface LeaderboardSummary {
  total_value: number
  total_users: number
}

export interface LeaderboardChartItem {
  username: string
  value: number
}

export interface LeaderboardData {
  items: LeaderboardEntry[]
  total: number
  page: number
  page_size: number
  pages: number
  summary?: LeaderboardSummary
  chart_items?: LeaderboardChartItem[]
}

export async function getBalanceLeaderboard(page = 1, pageSize = 10): Promise<LeaderboardData> {
  const { data } = await apiClient.get<LeaderboardData>('/public/leaderboard/balance', { params: { page, page_size: pageSize } })
  return data
}

export async function getConsumptionLeaderboard(period: 'daily' | 'weekly' | 'monthly' = 'daily', page = 1, pageSize = 10): Promise<LeaderboardData> {
  const { data } = await apiClient.get<LeaderboardData>('/public/leaderboard/consumption', { params: { period, page, page_size: pageSize } })
  return data
}

export async function getCheckinLeaderboard(page = 1, pageSize = 10): Promise<LeaderboardData> {
  const { data } = await apiClient.get<LeaderboardData>('/public/leaderboard/checkin', { params: { page, page_size: pageSize } })
  return data
}

export async function getTransferLeaderboard(period = 'day', page = 1, pageSize = 20): Promise<LeaderboardData> {
  const { data } = await apiClient.get<LeaderboardData>('/public/leaderboard/transfer', { params: { period, page, page_size: pageSize } })
  return data
}

export const leaderboardAPI = {
  getBalanceLeaderboard,
  getConsumptionLeaderboard,
  getCheckinLeaderboard,
  getTransferLeaderboard,
}
