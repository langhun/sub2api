import { describe, expect, it } from 'vitest'
import { openAIAccountActions } from '../accountActions'
import type { Account } from '@/types'

const account = (overrides: Partial<Account> = {}): Account => ({
  id: 1,
  name: 'OpenAI PAT',
  platform: 'openai',
  type: 'oauth',
  credentials: { openai_auth_mode: 'personal_access_token' },
  proxy_id: null,
  concurrency: 3,
  status: 'active',
  ...overrides,
})

describe('OpenAI AT account action', () => {
  it('recognizes all supported PAT mode spellings', () => {
    const action = openAIAccountActions[0]
    expect(action.visible(account())).toBe(true)
    expect(action.visible(account({ credentials: { auth_mode: 'personalAccessToken' } }))).toBe(true)
    expect(action.visible(account({ credentials: { auth_mode: 'PERSONAL_ACCESS_TOKEN' } }))).toBe(true)
    expect(action.visible(account({ credentials: { openai_auth_mode: ' Personal-Access-Token ' } }))).toBe(true)
  })

  it('is visible only for non-shadow OpenAI PAT OAuth accounts', () => {
    const action = openAIAccountActions[0]
    expect(action.visible(account({ type: 'apikey' }))).toBe(false)
    expect(action.visible(account({ platform: 'anthropic' }))).toBe(false)
    expect(action.visible(account({ parent_account_id: 9 }))).toBe(false)
    expect(action.visible(account({ credentials: { auth_mode: 'oauth' } }))).toBe(false)
  })
})
