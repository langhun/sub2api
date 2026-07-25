import { apiClient } from '@/api/client'

export type RewardDeliveryStatus = 'pending' | 'delivering' | 'delivered' | 'failed' | 'compensated'

export interface RewardDelivery {
  id: number
  source_type: string
  source_id: number
  user_id: number
  prize_item_id?: number
  reward_type: string
  reward_value: number
  reward_detail: string
  rule_version: string
  status: RewardDeliveryStatus
  attempts: number
  last_error?: string
  next_retry_at?: string
  delivered_at?: string
  compensated_at?: string
  created_at: string
  updated_at: string
}

export interface RewardDeliveryListParams {
  status?: RewardDeliveryStatus | ''
  source_type?: string
  user_id?: number
  page?: number
  page_size?: number
}

export interface RewardDeliveryList {
  items: RewardDelivery[]
  total: number
  page: number
  page_size: number
}

export async function listRewardDeliveries(params: RewardDeliveryListParams): Promise<RewardDeliveryList> {
  const { data } = await apiClient.get<RewardDeliveryList>('/admin/reward-deliveries', { params })
  return data
}

export async function retryRewardDelivery(id: number): Promise<RewardDelivery> {
  const { data } = await apiClient.post<RewardDelivery>(`/admin/reward-deliveries/${id}/retry`)
  return data
}

export async function compensateRewardDelivery(id: number, reason: string): Promise<RewardDelivery> {
  const { data } = await apiClient.post<RewardDelivery>(`/admin/reward-deliveries/${id}/compensate`, { reason })
  return data
}

export const rewardDeliveriesAPI = {
  list: listRewardDeliveries,
  retry: retryRewardDelivery,
  compensate: compensateRewardDelivery,
}

export default rewardDeliveriesAPI
