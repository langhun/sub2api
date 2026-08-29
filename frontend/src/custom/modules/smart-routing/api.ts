import { apiClient } from '@/api/client'

export interface SmartRoutingGroup {
  group_id: number
  sort_order: number
  enabled: boolean
  name?: string
  platform?: string
  price: number
  success_rate: number
  speed_ms: number
  samples: number
  capacity_ratio: number
  score: number
  eligible: boolean
  reason?: string
}

export interface SmartRoutingProfile {
  api_key_id: number
  enabled: boolean
  mode: 'single' | 'smart'
  strategy: 'auto' | 'price' | 'speed' | 'success' | 'custom'
  price_weight: number
  speed_weight: number
  success_weight: number
  sticky_ttl_seconds: number
  exploration_rate: number
  version: number
  groups: SmartRoutingGroup[]
}

export interface SmartRoutingUpdate {
  /** Required optimistic-lock version; GET the profile first and pass profile.version. */
  version: number
  enabled: boolean
  mode: 'single' | 'smart'
  strategy: SmartRoutingProfile['strategy']
  price_weight: number
  speed_weight: number
  success_weight: number
  sticky_ttl_seconds: number
  exploration_rate: number
  group_ids: number[]
}

export async function getSmartRouting(keyId: number): Promise<SmartRoutingProfile> {
  const { data } = await apiClient.get<SmartRoutingProfile>(`/keys/${keyId}/routing`)
  return data
}

export async function updateSmartRouting(keyId: number, payload: SmartRoutingUpdate): Promise<SmartRoutingProfile> {
  const { data } = await apiClient.put<SmartRoutingProfile>(`/keys/${keyId}/routing`, payload)
  return data
}

export async function previewSmartRouting(keyId: number, model: string, endpoint = 'chat_completions') {
  const { data } = await apiClient.post<{ profile: SmartRoutingProfile; groups: SmartRoutingGroup[] }>(
    `/keys/${keyId}/routing/preview`,
    { model, endpoint }
  )
  return data
}
