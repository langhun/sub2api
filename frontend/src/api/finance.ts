import { apiClient } from './client'
import type { PaginatedResponse } from '@/types'

export interface FinanceAccount {
  account_id: number
  balance: string
  frozen_amount: string
  credit_limit: string
  debt_principal: string
  debt_interest: string
  total_debt: string
  available_capacity: string
  status: string
  legacy_missing: boolean
}

export interface FinanceTransaction {
  id: number
  tx_id: string
  user_id: number
  account_id: number
  tx_type: string
  business_module: string
  amount: string
  balance_before: string
  balance_after: string
  frozen_before: string
  frozen_after: string
  credit_limit_snapshot: string
  debt_snapshot: string
  description: string
  reference_type?: string
  reference_id?: string
  request_id?: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface FinanceTransactionQuery {
  page?: number
  page_size?: number
  type?: string
  business_module?: string
}

export type BankAccount = FinanceAccount
export type BankTransaction = FinanceTransaction
export type BankTransactionQuery = FinanceTransactionQuery

export async function getFinanceAccount(): Promise<FinanceAccount> {
  const { data } = await apiClient.get<FinanceAccount>('/finance/account')
  return data
}

export async function getFinanceTransactions(
  params: FinanceTransactionQuery = {},
): Promise<PaginatedResponse<FinanceTransaction>> {
  const { data } = await apiClient.get<PaginatedResponse<FinanceTransaction>>('/finance/transactions', { params })
  return data
}

export const financeAPI = {
  getFinanceAccount,
  getFinanceTransactions,
}

// 渐进式收口期间保留 bankAPI 别名，避免历史 import 一次性失效。
export const bankAPI = financeAPI

export default financeAPI
