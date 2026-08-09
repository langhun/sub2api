import type { Account } from '@/types'

export interface CustomAccountAction {
  id: string
  labelKey: string
  icon: 'key'
  visible: (account: Account) => boolean
}

const isOpenAIPAT = (account: Account) => {
  if (account.platform !== 'openai' || account.type !== 'oauth' || account.parent_account_id != null) return false
  const credentials = account.credentials as Record<string, unknown> | undefined
  return [credentials?.openai_auth_mode, credentials?.auth_mode].some(isPersonalAccessTokenMode)
}

// The backend accepts both the legacy snake_case marker and the current
// camelCase marker. Normalize both forms so the action remains visible for
// accounts created by either import path without exposing the token itself.
const isPersonalAccessTokenMode = (value: unknown): boolean => {
  if (typeof value !== 'string') return false
  return value.trim().toLowerCase().replace(/[_\s-]/g, '') === 'personalaccesstoken'
}

export const openAIAccountActions: readonly CustomAccountAction[] = [
  { id: 'openai-at', labelKey: 'admin.openaiAt.manage', icon: 'key', visible: isOpenAIPAT }
]
