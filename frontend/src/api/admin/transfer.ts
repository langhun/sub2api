import { apiClient } from '../client'
import type { TransferRecord } from './transfer'

export interface DailyFeeStat {
  date: string
  total_fee: number
  count: number
}

export async function listTransfers(params: {
  page?: number
  page_size?: number
  status?: string
  transfer_type?: string
  user_id?: number
  start_time?: string
  end_time?: string
}): Promise<{ items: TransferRecord[]; total: number; page: number; page_size: number }> {
  const { data } = await apiClient.get('/admin/transfers', { params })
  return data
}

export async function freezeTransfer(id: number): Promise<void> {
  await apiClient.put(`/admin/transfers/${id}/freeze`)
}

export async function revokeTransfer(id: number, reason: string): Promise<void> {
  await apiClient.put(`/admin/transfers/${id}/revoke`, { reason })
}

export async function batchDistribute(
  targets: { user_id: number; amount: number }[],
  memo?: string,
): Promise<{ items: TransferRecord[]; count: number }> {
  const { data } = await apiClient.post('/admin/transfers/batch', { targets, memo })
  return data
}

export async function getFeeStats(params: {
  start_time?: string
  end_time?: string
}): Promise<DailyFeeStat[]> {
  const { data } = await apiClient.get('/admin/transfers/stats', { params })
  return data
}

export async function listRedPackets(params: {
  page?: number
  page_size?: number
}): Promise<{ items: any[]; total: number; page: number; page_size: number }> {
  const { data } = await apiClient.get('/admin/redpackets', { params })
  return data
}
