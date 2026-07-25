import { apiClient } from '@/api/client'
import { createActivityIdempotencyKey } from './idempotency'

export interface RedPacketRecord {
  id: number
  sender_id: number
  total_amount: number
  total_count: number
  remaining_amount: number
  remaining_count: number
  redpacket_type: 'equal' | 'random'
  fee: number
  fee_rate: number
  code: string
  status: 'active' | 'expired' | 'exhausted'
  memo: string | null
  expire_at: string
  created_at: string
}

export interface RedPacketClaimRecord {
  id: number
  redpacket_id: number
  user_id: number
  user_display: string
  amount: number
  transfer_id: number | null
  created_at: string
}

export async function createRedPacket(params: {
  total_amount: number
  count: number
  redpacket_type?: 'equal' | 'random'
  memo?: string
}, requestKey?: string): Promise<RedPacketRecord> {
  const { data } = await apiClient.post<RedPacketRecord>('/redpacket', params, { headers: { 'Idempotency-Key': requestKey || createActivityIdempotencyKey('redpacket-create') } })
  return data
}

export async function claimRedPacket(code: string, requestKey?: string): Promise<RedPacketClaimRecord> {
  const { data } = await apiClient.post<RedPacketClaimRecord>('/redpacket/claim', { code }, { headers: { 'Idempotency-Key': requestKey || createActivityIdempotencyKey('redpacket-claim') } })
  return data
}

export async function getRedPacketDetail(id: number): Promise<{
  redpacket: RedPacketRecord
  claims: RedPacketClaimRecord[]
}> {
  const { data } = await apiClient.get(`/redpacket/${id}`)
  return data
}

export async function getMyRedPackets(params: {
  role?: 'sent' | 'received'
  page?: number
  page_size?: number
}): Promise<{ items: RedPacketRecord[]; total: number; page: number; page_size: number }> {
  const { data } = await apiClient.get('/redpacket/my', { params })
  return data
}

export const redPacketAPI = {
  createRedPacket,
  claimRedPacket,
  getRedPacketDetail,
  getMyRedPackets,
}

export default redPacketAPI
