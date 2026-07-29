import { apiClient } from '@/api/client'

export interface AccountDrainTargetStatus {
  account_id: number
  active: boolean
}

export async function getAccountDrainTargetStatus(accountID: number): Promise<AccountDrainTargetStatus> {
  const { data } = await apiClient.get<AccountDrainTargetStatus>(`/admin/account-drain/accounts/${accountID}/status`)
  return data
}

export async function enableAccountDrainTarget(accountID: number): Promise<AccountDrainTargetStatus> {
  const { data } = await apiClient.put<AccountDrainTargetStatus>(`/admin/account-drain/accounts/${accountID}/target`)
  return data
}

export async function disableAccountDrainTarget(accountID: number): Promise<void> {
  await apiClient.delete(`/admin/account-drain/accounts/${accountID}/target`)
}
