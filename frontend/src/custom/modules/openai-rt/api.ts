import { apiClient } from '@/api/client'
import type { Account } from '@/types'

export interface OpenAIRTRevealResponse {
  account_id: number
  refresh_token: string
  masked_refresh_token: string
}

export interface OpenAIRTReplaceResponse {
  account: Account
  masked_refresh_token: string
}

export async function getOpenAIRT(accountID: number): Promise<OpenAIRTRevealResponse> {
  const { data } = await apiClient.get<OpenAIRTRevealResponse>(
    `/custom/admin/openai/accounts/${accountID}/refresh-token`,
    { headers: { 'Cache-Control': 'no-cache' } }
  )
  return data
}

export async function replaceOpenAIRT(accountID: number, refreshToken: string): Promise<OpenAIRTReplaceResponse> {
  const { data } = await apiClient.post<OpenAIRTReplaceResponse>(
    `/custom/admin/openai/accounts/${accountID}/refresh-token`,
    { refresh_token: refreshToken },
    { timeout: 90_000 }
  )
  return data
}
