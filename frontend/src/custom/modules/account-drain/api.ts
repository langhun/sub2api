import { apiClient } from '@/api/client'

export type AccountDrainPlanStatus = 'active' | 'stopped' | 'expired'

export interface AccountDrainPlan {
  id: number
  name: string
  status: AccountDrainPlanStatus
  expires_at: string | null
  account_ids: number[]
  created_at: string
  updated_at: string
}

export interface CreateAccountDrainPlanInput {
  name: string
  account_ids: number[]
  expires_at?: string | null
}

export async function listAccountDrainPlans(): Promise<AccountDrainPlan[]> {
  const { data } = await apiClient.get<AccountDrainPlan[]>('/admin/account-drain/plans')
  return data ?? []
}

export async function createAccountDrainPlan(input: CreateAccountDrainPlanInput): Promise<AccountDrainPlan> {
  const { data } = await apiClient.post<AccountDrainPlan>('/admin/account-drain/plans', input)
  return data
}

export async function stopAccountDrainPlan(id: number): Promise<void> {
  await apiClient.post(`/admin/account-drain/plans/${id}/stop`)
}
