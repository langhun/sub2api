import { apiClient } from '@/api/client'
import type { Account } from '@/types'

export interface OpenAIATRevealResponse {
  account_id: number
  access_token: string
  masked_access_token: string
}

export interface OpenAIATReplaceResponse {
  account: Account
  masked_access_token: string
}

export async function getOpenAIAT(accountID: number): Promise<OpenAIATRevealResponse> {
  const { data } = await apiClient.get<OpenAIATRevealResponse>(
    `/custom/admin/openai/accounts/${accountID}/access-token`,
    { headers: { 'Cache-Control': 'no-cache' } }
  )
  return data
}

export async function replaceOpenAIAT(accountID: number, accessToken: string): Promise<OpenAIATReplaceResponse> {
  const { data } = await apiClient.post<OpenAIATReplaceResponse>(
    `/custom/admin/openai/accounts/${accountID}/access-token`,
    { access_token: accessToken }
  )
  return data
}
