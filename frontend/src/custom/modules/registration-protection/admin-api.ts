import { apiClient } from '@/api/client'

export type RegistrationRiskAction = 'none' | 'observe' | 'medium' | 'high' | 'manual_lock' | 'manual_unlock'

export interface RegistrationRiskEvidence {
  signal: string
  detail: string
}

export interface RegistrationRiskItem {
  user_id: number
  email: string
  score: number
  action: RegistrationRiskAction | string
  signals: string[]
  /** Added by the risk API; optional for compatibility with older responses. */
  evidence?: RegistrationRiskEvidence[]
  expires_at: string | null
  payment_unlocked: boolean
}

export interface RegistrationRiskListResponse {
  items: RegistrationRiskItem[]
  has_more?: boolean
}

export interface RegistrationRiskOverrideResponse {
  user_id: number
  action: RegistrationRiskAction | string
}

export async function listRegistrationRisk(limit = 100, offset = 0): Promise<RegistrationRiskListResponse> {
  const { data } = await apiClient.get<RegistrationRiskListResponse>('/custom/admin/registration-risk', {
    params: { limit, ...(offset > 0 ? { offset } : {}) },
  })
  return data
}

export async function unlockRegistrationRisk(userID: number, reason: string): Promise<RegistrationRiskOverrideResponse> {
  const { data } = await apiClient.post<RegistrationRiskOverrideResponse>(
    `/custom/admin/registration-risk/${userID}/unlock`,
    { reason },
  )
  return data
}

export async function lockRegistrationRisk(userID: number, reason: string): Promise<RegistrationRiskOverrideResponse> {
  const { data } = await apiClient.post<RegistrationRiskOverrideResponse>(
    `/custom/admin/registration-risk/${userID}/lock`,
    { reason },
  )
  return data
}

const registrationRiskAPI = {
  list: listRegistrationRisk,
  unlock: unlockRegistrationRisk,
  lock: lockRegistrationRisk,
}

export default registrationRiskAPI
