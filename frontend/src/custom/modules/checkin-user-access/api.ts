import { apiClient } from '@/api/client'

export interface CheckinUserAccess {
  user_id: number
  disabled: boolean
  reason?: string
  updated_by?: number
  updated_at?: string
}

export const checkinUserAccessAPI = {
  get: (userID: number) => apiClient.get<CheckinUserAccess>(`/custom/admin/checkin/users/${userID}/access`),
  setDisabled: (userID: number, disabled: boolean) => apiClient.put<CheckinUserAccess>(
    `/custom/admin/checkin/users/${userID}/access`,
    { disabled },
  ),
}
