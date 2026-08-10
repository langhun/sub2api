import { describe, expect, it } from 'vitest'
import { openAIAccountActions } from '../accountActions'
import type { Account } from '@/types'

const account = (overrides: Partial<Account> = {}): Account => ({
  id: 1,
  name: 'OpenAI OAuth',
  platform: 'openai',
  type: 'oauth',
  credentials: { auth_mode: 'oauth' },
  credentials_status: { has_refresh_token: true },
  proxy_id: null,
  concurrency: 3,
  status: 'active',
  ...overrides,
})

describe('OpenAI RT account action', () => {
  it('is visible for a normal OpenAI OAuth account with a refresh token', () => {
    const action = openAIAccountActions[0]
    expect(action.visible(account())).toBe(true)
  })

  it('hides the action for PAT, API key, missing RT, and shadow accounts', () => {
    const action = openAIAccountActions[0]
    expect(action.visible(account({ type: 'apikey' }))).toBe(false)
    expect(action.visible(account({ platform: 'anthropic' }))).toBe(false)
    expect(action.visible(account({ parent_account_id: 9 }))).toBe(false)
    expect(action.visible(account({ credentials_status: { has_refresh_token: false } }))).toBe(false)
    expect(action.visible(account({ credentials: { auth_mode: 'personalAccessToken' } }))).toBe(false)
    expect(action.visible(account({ credentials: { openai_auth_mode: 'PERSONAL_ACCESS_TOKEN' } }))).toBe(false)
  })
})
