import { apiClient } from '@/api/client'

export interface TransferLeaderboardEntry {
  rank: number
  user_id: number
  email: string
  display_name: string
  total_amount: number
  total_count: number
}

export async function getTransferLeaderboard(params: {
  period?: string
  limit?: number
}): Promise<TransferLeaderboardEntry[]> {
  const { data } = await apiClient.get<TransferLeaderboardEntry[]>('/transfer/leaderboard', { params })
  return data
}
