import { describe, it, expect } from 'vitest'
import {
  matchesPlatform,
  matchesType,
  matchesGroup,
  matchesPrivacyMode,
  matchesSearch,
  matchesTier,
  getAntigravityTierFromAccount,
  ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE,
  ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE
} from '../accountFilters'
import type { Account } from '@/types'

describe('accountFilters', () => {
  const createMockAccount = (overrides: Partial<Account> = {}): Account => ({
    id: 1,
    name: 'Test Account',
    platform: 'openai',
    type: 'oauth',
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '2024-01-01',
    updated_at: '2024-01-01',
    proxy_id: null,
    concurrency: 1,
    priority: 0,
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    ...overrides
  })

  describe('matchesPlatform', () => {
    it('should return true when platform filter is empty', () => {
      const account = createMockAccount({ platform: 'openai' })
      expect(matchesPlatform(account, '')).toBe(true)
    })

    it('should return true when platform matches', () => {
      const account = createMockAccount({ platform: 'openai' })
      expect(matchesPlatform(account, 'openai')).toBe(true)
    })

    it('should return false when platform does not match', () => {
      const account = createMockAccount({ platform: 'openai' })
      expect(matchesPlatform(account, 'gemini')).toBe(false)
    })
  })

  describe('matchesType', () => {
    it('should return true when type filter is empty', () => {
      const account = createMockAccount({ type: 'oauth' })
      expect(matchesType(account, '')).toBe(true)
    })

    it('should return true when type matches', () => {
      const account = createMockAccount({ type: 'oauth' })
      expect(matchesType(account, 'oauth')).toBe(true)
    })

    it('should return false when type does not match', () => {
      const account = createMockAccount({ type: 'oauth' })
      expect(matchesType(account, 'api_key')).toBe(false)
    })
  })

  describe('matchesGroup', () => {
    it('should return true when group filter is empty', () => {
      const account = createMockAccount({ group_ids: [1, 2] })
      expect(matchesGroup(account, '')).toBe(true)
    })

    it('should return true when account has the group', () => {
      const account = createMockAccount({ group_ids: [1, 2, 3] })
      expect(matchesGroup(account, '2')).toBe(true)
    })

    it('should return false when account does not have the group', () => {
      const account = createMockAccount({ group_ids: [1, 2] })
      expect(matchesGroup(account, '5')).toBe(false)
    })

    it('should match ungrouped accounts', () => {
      const account = createMockAccount({ group_ids: [] })
      expect(matchesGroup(account, ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE)).toBe(true)
    })

    it('should not match grouped accounts as ungrouped', () => {
      const account = createMockAccount({ group_ids: [1] })
      expect(matchesGroup(account, ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE)).toBe(false)
    })

    it('should handle groups from groups array', () => {
      const account = createMockAccount({
        group_ids: undefined,
        groups: [{ id: 1 }, { id: 2 }] as any
      })
      expect(matchesGroup(account, '1')).toBe(true)
    })
  })

  describe('matchesPrivacyMode', () => {
    it('should return true when privacy mode filter is empty', () => {
      const account = createMockAccount({ extra: { privacy_mode: 'high' } })
      expect(matchesPrivacyMode(account, '')).toBe(true)
    })

    it('should match when privacy mode equals filter', () => {
      const account = createMockAccount({ extra: { privacy_mode: 'high' } })
      expect(matchesPrivacyMode(account, 'high')).toBe(true)
    })

    it('should not match when privacy mode differs', () => {
      const account = createMockAccount({ extra: { privacy_mode: 'high' } })
      expect(matchesPrivacyMode(account, 'low')).toBe(false)
    })

    it('should match unset privacy mode', () => {
      const account = createMockAccount({ extra: { privacy_mode: '' } })
      expect(matchesPrivacyMode(account, ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE)).toBe(true)
    })

    it('should match when extra is undefined', () => {
      const account = createMockAccount({ extra: undefined })
      expect(matchesPrivacyMode(account, ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE)).toBe(true)
    })
  })

  describe('matchesSearch', () => {
    it('should return true when search is empty', () => {
      const account = createMockAccount({ name: 'Test Account' })
      expect(matchesSearch(account, '')).toBe(true)
    })

    it('should match case-insensitive substring', () => {
      const account = createMockAccount({ name: 'Test Account' })
      expect(matchesSearch(account, 'test')).toBe(true)
      expect(matchesSearch(account, 'TEST')).toBe(true)
      expect(matchesSearch(account, 'account')).toBe(true)
    })

    it('should not match when search term is not found', () => {
      const account = createMockAccount({ name: 'Test Account' })
      expect(matchesSearch(account, 'xyz')).toBe(false)
    })

    it('should trim search term', () => {
      const account = createMockAccount({ name: 'Test Account' })
      expect(matchesSearch(account, '  test  ')).toBe(true)
    })
  })

  describe('matchesTier', () => {
    it('should return true when tier filter is empty', () => {
      const account = createMockAccount()
      expect(matchesTier(account, '', 'openai')).toBe(true)
    })

    it('should match OpenAI tier', () => {
      const account = createMockAccount({
        platform: 'openai',
        credentials: { plan_type: 'plus' }
      })
      expect(matchesTier(account, 'plus', 'openai')).toBe(true)
    })

    it('should normalize OpenAI tier aliases', () => {
      const account = createMockAccount({
        platform: 'openai',
        credentials: { plan_type: 'chatgpt_plus' }
      })
      expect(matchesTier(account, 'plus', 'openai')).toBe(true)
    })

    it('should match Gemini tier', () => {
      const account = createMockAccount({
        platform: 'gemini',
        credentials: { tier_id: 'google_ai_pro' }
      })
      expect(matchesTier(account, 'google_ai_pro', 'gemini')).toBe(true)
    })

    it('should normalize Gemini tier aliases', () => {
      const account = createMockAccount({
        platform: 'gemini',
        credentials: { tier_id: 'ai_premium' }
      })
      expect(matchesTier(account, 'pro', 'gemini')).toBe(true)
    })

    it('should match Antigravity tier from credentials', () => {
      const account = createMockAccount({
        platform: 'antigravity',
        credentials: { plan_type: 'pro' }
      })
      expect(matchesTier(account, 'g1-pro-tier', 'antigravity')).toBe(true)
    })

    it('should parse tier with platform prefix', () => {
      const account = createMockAccount({
        platform: 'openai',
        credentials: { plan_type: 'plus' }
      })
      expect(matchesTier(account, 'openai:plus', 'gemini')).toBe(true)
    })

    it('should not match when platform differs', () => {
      const account = createMockAccount({
        platform: 'openai',
        credentials: { plan_type: 'plus' }
      })
      expect(matchesTier(account, 'gemini:plus', 'openai')).toBe(false)
    })
  })

  describe('getAntigravityTierFromAccount', () => {
    it('should return null for non-antigravity platform', () => {
      const account = createMockAccount({ platform: 'openai' })
      expect(getAntigravityTierFromAccount(account)).toBeNull()
    })

    it('should return tier from credentials plan_type', () => {
      const account = createMockAccount({
        platform: 'antigravity',
        credentials: { plan_type: 'pro' }
      })
      expect(getAntigravityTierFromAccount(account)).toBe('g1-pro-tier')
    })

    it('should return tier from extra.load_code_assist.paidTier', () => {
      const account = createMockAccount({
        platform: 'antigravity',
        extra: {
          load_code_assist: {
            paidTier: { id: 'custom-tier' }
          }
        }
      })
      expect(getAntigravityTierFromAccount(account)).toBe('custom-tier')
    })

    it('should return tier from extra.load_code_assist.currentTier', () => {
      const account = createMockAccount({
        platform: 'antigravity',
        extra: {
          load_code_assist: {
            currentTier: { id: 'current-tier' }
          }
        }
      })
      expect(getAntigravityTierFromAccount(account)).toBe('current-tier')
    })

    it('should return null when no tier found', () => {
      const account = createMockAccount({
        platform: 'antigravity',
        credentials: {}
      })
      expect(getAntigravityTierFromAccount(account)).toBeNull()
    })
  })
})




