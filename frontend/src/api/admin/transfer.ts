import { apiClient } from '../client'
import type { TransferRecord } from '../transfer'

export type { TransferRecord }

export interface DailyFeeStat {
  date: string
  total_fee: number
  count: number
}

const adminTransferAPI = {
  listTransfers(params: {
    page?: number
    page_size?: number
    status?: string
    transfer_type?: string
    user_id?: number
    start_time?: string
    end_time?: string
  }) {
    return apiClient.get<{ items: TransferRecord[]; total: number; page: number; page_size: number }>('/admin/transfers', { params })
      .then(res => res.data)
  },

  freezeTransfer(id: number) {
    return apiClient.put(`/admin/transfers/${id}/freeze`)
  },

  revokeTransfer(id: number, reason: string) {
    return apiClient.put(`/admin/transfers/${id}/revoke`, { reason })
  },

  batchDistribute(targets: { user_id: number; amount: number }[], memo?: string) {
    return apiClient.post<{ items: TransferRecord[]; count: number }>('/admin/transfers/batch', { targets, memo })
      .then(res => res.data)
  },

  getFeeStats(params: { start_time?: string; end_time?: string }) {
    return apiClient.get<DailyFeeStat[]>('/admin/transfers/stats', { params })
      .then(res => res.data)
  },

  listRedPackets(params: { page?: number; page_size?: number }) {
    return apiClient.get<{ items: any[]; total: number; page: number; page_size: number }>('/admin/redpackets', { params })
      .then(res => res.data)
  },
}

export default adminTransferAPI
