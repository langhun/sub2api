import type { BasePaginationResponse } from '@/types'
import { apiClient } from '@/api/client'

export type RechargePromotionMode = 'percentage' | 'tiered'

export interface RechargePromotionTierRule {
  minimum_amount: number
  bonus_credit: number
}

export interface RechargePromotionPreview {
  active: boolean
  mode: RechargePromotionMode
  base_credit: number
  bonus_credit: number
  final_credit: number
  ineligible_reason: string
  minimum_amount: number
  percentage_rate: number
  maximum_bonus: number
  tier_rules: RechargePromotionTierRule[]
  campaign_started_at: string
  campaign_ended_at: string
}

export interface RechargePromotionOrder {
  payment_order_id: number
  user_id: number
  mode: RechargePromotionMode
  payment_amount: number
  pay_amount: number
  base_credit: number
  bonus_credit: number
  final_credit: number
  refunded_credit: number
  refunded_base_credit: number
  refunded_bonus_credit: number
  created_at: string
}

export interface RechargePromotionSummary {
  enabled: boolean
  mode: RechargePromotionMode
  orders: number
  bonus_credit: number
  final_credit: number
  refunded_bonus_credit: number
}

export const rechargePromotionsAPI = {
  preview(amount: number, baseCredit: number) {
    return apiClient.post<RechargePromotionPreview>('/recharge-promotions/preview', {
      amount,
      base_credit: baseCredit,
    })
  },

  getUserOrder(orderID: number) {
    return apiClient.get<RechargePromotionOrder>(`/recharge-promotions/orders/${orderID}`)
  },

  getAdminOrder(orderID: number) {
    return apiClient.get<RechargePromotionOrder>(`/custom/admin/recharge-promotions/orders/${orderID}`)
  },

  getSummary() {
    return apiClient.get<RechargePromotionSummary>('/custom/admin/recharge-promotions/summary')
  },

  getAdminOrders(params?: { page?: number; page_size?: number; mode?: RechargePromotionMode }) {
    return apiClient.get<BasePaginationResponse<RechargePromotionOrder>>('/custom/admin/recharge-promotions/orders', { params })
  },
}
