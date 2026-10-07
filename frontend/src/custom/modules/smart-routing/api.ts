import { apiClient } from '@/api/client'

export const MAX_EXPLORATION_RATE = 0.01

export class VersionConflictError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'VersionConflictError'
  }
}

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
  price_score?: number
  speed_score?: number
  success_score?: number
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
  const normalizedPayload = {
    ...payload,
    exploration_rate: Math.min(payload.exploration_rate, MAX_EXPLORATION_RATE)
  }
  try {
    const { data } = await apiClient.put<SmartRoutingProfile>(`/keys/${keyId}/routing`, normalizedPayload)
    return data
  } catch (err: any) {
    if (err.response?.status === 400) {
      const detail = err.response?.data?.detail || ''
      if (typeof detail === 'string' && detail.toLowerCase().includes('version conflict')) {
        throw new VersionConflictError('Configuration has been modified by another user. Please refresh and try again.')
      }
    }
    throw err
  }
}

export async function previewSmartRouting(keyId: number, model: string, endpoint = 'chat_completions') {
  const { data } = await apiClient.post<{ profile: SmartRoutingProfile; groups: SmartRoutingGroup[] }>(
    `/keys/${keyId}/routing/preview`,
    { model, endpoint }
  )
  return data
}
