import { apiClient } from './client'

export interface LotteryCurrent {
  lottery_type: string
  issue_no: string
  open_time: string
  cutoff_time: string
  is_closed: boolean
  jackpot_balance: string
}

export interface LotteryBetPayload {
  red_balls: number[]
  blue_ball: number
}

export interface LotteryBetResult {
  order_id?: number
  order_ids?: number[]
  issue_no: string
  lottery_type: string
  cost: string
  status: string
  created_at?: string
}

export interface LotteryOrder {
  order_id: number
  issue_no: string
  lottery_type: string
  red_balls: string[]
  blue_ball: string
  cost: string
  status: string
  reward: string
  prize_level: string
  red_hits: number
  blue_hit: boolean
  created_at: string
}

export interface LotteryResult {
  issue_no: string
  lottery_type: string
  red_balls: string[]
  blue_ball: string
  opened_at: string
  source: string
  source_ref: string
  created_at: string
}

function buildIdempotencyKey(payload: LotteryBetPayload, issueNo?: string): string {
  const numbers = [...payload.red_balls].sort((a, b) => a - b).join('-')
  const prefix = issueNo ? `lottery:${issueNo}` : 'lottery'
  const randomPart =
    typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `${prefix}:${numbers}:${payload.blue_ball}:${randomPart}`
}

export const lotteryAPI = {
  async getCurrent(): Promise<LotteryCurrent> {
    const { data } = await apiClient.get<LotteryCurrent>('/lottery/current')
    return data
  },

  async createBet(payload: LotteryBetPayload, issueNo?: string): Promise<LotteryBetResult> {
    const { data } = await apiClient.post<LotteryBetResult>('/lottery/bet', payload, {
      headers: {
        'Idempotency-Key': buildIdempotencyKey(payload, issueNo),
      },
    })
    return data
  },

  async getOrders(issueNo?: string): Promise<LotteryOrder[]> {
    const { data } = await apiClient.get<LotteryOrder[]>('/lottery/orders', {
      params: issueNo ? { issue_no: issueNo } : undefined,
    })
    return data
  },

  async getResults(limit?: number): Promise<LotteryResult[]> {
    const { data } = await apiClient.get<LotteryResult[]>('/lottery/results', {
      params: limit ? { limit } : undefined,
    })
    return data
  },

  async getResult(issueNo: string): Promise<LotteryResult> {
    const { data } = await apiClient.get<LotteryResult>(`/lottery/results/${issueNo}`)
    return data
  },
}

export default lotteryAPI
