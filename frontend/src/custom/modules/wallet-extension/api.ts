import { apiClient } from '../../../api/client'

export interface DirectTransferRecord {
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

export interface DirectTransferStats {
  total_sent: number
  total_received: number
  total_fee_paid: number
}

export interface DirectTransferValidation {
  fee: number
  fee_rate: number
  gross_amount?: number
  receiver_display?: string
  receiver_id?: number
  daily_remaining_amount?: number
  daily_remaining_count?: number
}

export interface DirectTransferReceiver {
  receiver_id: number
  receiver_display: string
  receiver_username?: string
  receiver_email?: string
}

export interface DirectTransferHistoryParams {
  role?: 'sender' | 'receiver'
  page?: number
  page_size?: number
}

export interface DirectTransferHistory {
  items: DirectTransferRecord[]
  total: number
  page: number
  page_size: number
}

export function createDirectTransferIdempotencyKey(scope: string): string {
  const suffix = typeof crypto !== 'undefined' && crypto.randomUUID
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `${scope}-${suffix}`
}

export async function searchDirectTransferReceivers(query: string): Promise<DirectTransferReceiver[]> {
  const { data } = await apiClient.get<DirectTransferReceiver[]>('/transfer/receivers', {
    // Do not reuse a masked response after the display policy changes.
    params: { query, _t: Date.now() },
  })
  return data
}

export async function resolveDirectTransferReceiver(query: string): Promise<DirectTransferReceiver> {
  const { data } = await apiClient.get<DirectTransferReceiver>('/transfer/receiver', {
    params: { query },
  })
  return data
}

export async function submitDirectTransfer(
  receiverId: number,
  amount: number,
  memo?: string,
  requestKey?: string,
): Promise<DirectTransferRecord> {
  const { data } = await apiClient.post<DirectTransferRecord>('/transfer', {
    receiver_id: receiverId,
    amount,
    memo,
  }, {
    headers: {
      'Idempotency-Key': requestKey || createDirectTransferIdempotencyKey('balance-transfer'),
    },
  })
  return data
}

export async function validateDirectTransfer(
  receiverId: number,
  amount: number,
  requestKey?: string,
): Promise<DirectTransferValidation> {
  const { data } = await apiClient.post<DirectTransferValidation>('/transfer/validate', {
    receiver_id: receiverId,
    amount,
  }, {
    headers: {
      'Idempotency-Key': requestKey || createDirectTransferIdempotencyKey('transfer-validate'),
    },
  })
  return data
}

export async function getDirectTransferHistory(
  params: DirectTransferHistoryParams,
): Promise<DirectTransferHistory> {
  const { data } = await apiClient.get<DirectTransferHistory>('/transfer/history', { params })
  return data
}

export async function getDirectTransferStats(): Promise<DirectTransferStats> {
  const { data } = await apiClient.get<DirectTransferStats>('/transfer/stats')
  return data
}

export const directTransferAPI = {
  searchTransferReceivers: searchDirectTransferReceivers,
  resolveTransferReceiver: resolveDirectTransferReceiver,
  transferBalance: submitDirectTransfer,
  validateTransfer: validateDirectTransfer,
  getTransferHistory: getDirectTransferHistory,
  getTransferStats: getDirectTransferStats,
}
