import { apiClient } from '@/api/client'
import type { DirectTransferRecord } from '../api'

export interface DailyFeeStat {
  date: string
  total_fee: number
  count: number
}

export interface AdminTransferListParams {
  page?: number
  page_size?: number
  status?: string
  transfer_type?: string
  user_id?: number
  start_time?: string
  end_time?: string
}

export interface AdminTransferList {
  items: DirectTransferRecord[]
  total: number
  page: number
  page_size: number
}

export async function listAdminTransfers(params: AdminTransferListParams): Promise<AdminTransferList> {
  const { data } = await apiClient.get<AdminTransferList>('/admin/transfers', { params })
  return data
}

export async function freezeAdminTransfer(id: number): Promise<void> {
  await apiClient.put(`/admin/transfers/${id}/freeze`)
}

export async function revokeAdminTransfer(id: number, reason: string): Promise<void> {
  await apiClient.put(`/admin/transfers/${id}/revoke`, { reason })
}

export async function batchDistributeTransfers(
  targets: { user_id: number; amount: number }[],
  memo?: string,
): Promise<{ items: DirectTransferRecord[]; count: number }> {
  const { data } = await apiClient.post('/admin/transfers/batch', { targets, memo })
  return data
}

export async function getAdminTransferFeeStats(params: {
  start_time?: string
  end_time?: string
}): Promise<DailyFeeStat[]> {
  const { data } = await apiClient.get<DailyFeeStat[]>('/admin/transfers/stats', { params })
  return data
}

export async function listAdminRedPackets(params: {
  page?: number
  page_size?: number
}): Promise<{ items: unknown[]; total: number; page: number; page_size: number }> {
  const { data } = await apiClient.get('/admin/redpackets', { params })
  return data
}

export const adminTransferAPI = {
  listTransfers: listAdminTransfers,
  freezeTransfer: freezeAdminTransfer,
  revokeTransfer: revokeAdminTransfer,
  batchDistribute: batchDistributeTransfers,
  getFeeStats: getAdminTransferFeeStats,
  listRedPackets: listAdminRedPackets,
}
