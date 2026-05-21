import { apiClient } from '../client'

export interface PrizeItem {
  id: number
  name: string
  rarity: string
  reward_type: string
  reward_value: number
  reward_value_max: number
  subscription_id?: number | null
  subscription_days: number
  weight: number
  is_enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreatePrizeItemRequest {
  name: string
  rarity: string
  reward_type: string
  reward_value?: number
  reward_value_max?: number
  subscription_id?: number | null
  subscription_days?: number
  weight?: number
  is_enabled?: boolean
}

export interface UpdatePrizeItemRequest {
  name?: string
  rarity?: string
  reward_type?: string
  reward_value?: number
  reward_value_max?: number
  subscription_id?: number | null
  subscription_days?: number
  weight?: number
  is_enabled?: boolean
}

export async function listPrizeItems(): Promise<PrizeItem[]> {
  const { data } = await apiClient.get<PrizeItem[]>('/admin/blindbox/prize-items')
  return data
}

export async function createPrizeItem(req: CreatePrizeItemRequest): Promise<PrizeItem> {
  const { data } = await apiClient.post<PrizeItem>('/admin/blindbox/prize-items', req)
  return data
}

export async function updatePrizeItem(id: number, req: UpdatePrizeItemRequest): Promise<PrizeItem> {
  const { data } = await apiClient.put<PrizeItem>(`/admin/blindbox/prize-items/${id}`, req)
  return data
}

export async function deletePrizeItem(id: number): Promise<void> {
  await apiClient.delete(`/admin/blindbox/prize-items/${id}`)
}

export async function getBlindboxStats(): Promise<{ total_items: number; enabled_items: number; total_draws: number }> {
  const { data } = await apiClient.get('/admin/blindbox/stats')
  return data
}

export const blindboxAPI = {
  listPrizeItems,
  createPrizeItem,
  updatePrizeItem,
  deletePrizeItem,
  getBlindboxStats
}
