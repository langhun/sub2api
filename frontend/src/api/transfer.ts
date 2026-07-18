import { apiClient } from './client'

export function createActivityIdempotencyKey(scope: string): string {
  const suffix = typeof crypto !== 'undefined' && crypto.randomUUID
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `${scope}-${suffix}`
}

export interface TransferRecord {
  id: number
  sender_id: number
  receiver_id: number
  sender_display: string
  receiver_display: string
  amount: number
  fee: number
  fee_rate: number
  gross_amount: number
  transfer_type: 'direct' | 'redpacket' | 'batch'
  status: 'completed' | 'frozen' | 'revoked'
  memo: string | null
  redpacket_id: number | null
  created_at: string
}

export interface TransferStats {
  total_sent: number
  total_received: number
  total_fee_paid: number
}

export interface TransferLeaderboardEntry {
  rank: number
  user_id: number
  email: string
  display_name: string
  total_amount: number
  total_count: number
}

export interface TransferValidation {
  fee: number
  fee_rate: number
  gross_amount?: number
  receiver_display?: string
  receiver_id?: number
  daily_remaining_amount?: number
  daily_remaining_count?: number
}

export interface TransferReceiver {
  receiver_id: number
  receiver_display: string
  receiver_username?: string
  receiver_email?: string
}

export async function searchTransferReceivers(query: string): Promise<TransferReceiver[]> {
  const { data } = await apiClient.get<TransferReceiver[]>('/transfer/receivers', {
    // Avoid reusing an older masked GET response after the display policy changes.
    params: { query, _t: Date.now() },
  })
  return data
}

export async function resolveTransferReceiver(query: string): Promise<TransferReceiver> {
  const { data } = await apiClient.get<TransferReceiver>('/transfer/receiver', {
    params: { query },
  })
  return data
}

export async function transferBalance(receiverId: number, amount: number, memo?: string, requestKey?: string): Promise<TransferRecord> {
  const { data } = await apiClient.post<TransferRecord>('/transfer', {
    receiver_id: receiverId,
    amount,
    memo,
  }, { headers: { 'Idempotency-Key': requestKey || createActivityIdempotencyKey('balance-transfer') } })
  return data
}

export async function validateTransfer(receiverId: number, amount: number): Promise<TransferValidation> {
  const { data } = await apiClient.post<TransferValidation>('/transfer/validate', {
    receiver_id: receiverId,
    amount,
  }, { headers: { 'Idempotency-Key': createActivityIdempotencyKey('transfer-validate') } })
  return data
}

export async function getTransferHistory(params: {
  role?: 'sender' | 'receiver'
  page?: number
  page_size?: number
}): Promise<{ items: TransferRecord[]; total: number; page: number; page_size: number }> {
  const { data } = await apiClient.get('/transfer/history', { params })
  return data
}

export async function getTransferStats(): Promise<TransferStats> {
  const { data } = await apiClient.get<TransferStats>('/transfer/stats')
  return data
}

export async function getTransferLeaderboard(params: {
  period?: string
  limit?: number
}): Promise<TransferLeaderboardEntry[]> {
  const { data } = await apiClient.get<TransferLeaderboardEntry[]>('/transfer/leaderboard', { params })
  return data
}

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

export const transferAPI = {
  searchTransferReceivers,
  resolveTransferReceiver,
  transferBalance,
  validateTransfer,
  getTransferHistory,
  getTransferStats,
  getTransferLeaderboard,
  createRedPacket,
  claimRedPacket,
  getRedPacketDetail,
  getMyRedPackets,
}

export default transferAPI
