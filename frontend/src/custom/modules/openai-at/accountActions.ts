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
  return credentials?.openai_auth_mode === 'personal_access_token' || credentials?.auth_mode === 'personalAccessToken'
}

export const openAIAccountActions: readonly CustomAccountAction[] = [
  { id: 'openai-at', labelKey: 'admin.openaiAt.manage', icon: 'key', visible: isOpenAIPAT }
]
