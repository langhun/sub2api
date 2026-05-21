import type { Account } from '@/types'
import {
  matchesAccountMainStatusFilter,
  matchesAccountRuntimeStatusFilter,
  matchesAccountSchedulingStatusFilter,
  type AccountMainStatusFilterValue,
  type AccountRuntimeStatusFilterValue,
  type AccountSchedulingStatusFilterValue
} from './accountStatus'

export const ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE = 'ungrouped'
export const ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE = '__unset__'

export interface AccountFilterOptions {
  platform?: string
  tier?: string
  type?: string
  main_status?: string
  runtime_status?: string
  scheduling_status?: string
  privacy_mode?: string
  group?: string
  search?: string
}

export interface StatusEvalOptions {
  nowMs: number
}

export function matchesPlatform(account: Account, platform: string): boolean {
  if (!platform) return true
  return account.platform === platform
}

export function matchesType(account: Account, type: string): boolean {
  if (!type) return true
  return account.type === type
}

export function matchesStatus(
  account: Account,
  mainStatus: string,
  runtimeStatus: string,
  schedulingStatus: string,
  options: StatusEvalOptions
): boolean {
  if (!matchesAccountMainStatusFilter(account, mainStatus as AccountMainStatusFilterValue)) {
    return false
  }
  if (!matchesAccountRuntimeStatusFilter(account, runtimeStatus as AccountRuntimeStatusFilterValue, options)) {
    return false
  }
  if (!matchesAccountSchedulingStatusFilter(account, schedulingStatus as AccountSchedulingStatusFilterValue, options)) {
    return false
  }
  return true
}

export function matchesGroup(account: Account, group: string): boolean {
  if (!group) return true

  const groupIds = account.group_ids ?? account.groups?.map((g) => g.id) ?? []

  if (group === ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE) {
    return groupIds.length === 0
  }

  return groupIds.includes(Number(group))
}

export function matchesPrivacyMode(account: Account, privacyMode: string): boolean {
  if (!privacyMode) return true

  const accountPrivacyMode = typeof account.extra?.privacy_mode === 'string'
    ? account.extra.privacy_mode
    : ''

  if (privacyMode === ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE) {
    return accountPrivacyMode.trim() === ''
  }

  return accountPrivacyMode === privacyMode
}

export function matchesSearch(account: Account, search: string): boolean {
  const searchTerm = String(search || '').trim().toLowerCase()
  if (!searchTerm) return true

  return account.name.toLowerCase().includes(searchTerm)
}

const OPENAI_TIER_ALIASES: Record<string, string> = {
  free: 'free',
  free_plan: 'free',
  chatgpt_free: 'free',
  plus: 'plus',
  plus_plan: 'plus',
  chatgpt_plus: 'plus',
  team: 'team',
  team_plan: 'team',
  chatgpt_team: 'team',
  business: 'team',
  pro: 'pro',
  pro_plan: 'pro',
  chatgpt_pro: 'pro',
  enterprise: 'enterprise',
  enterprise_plan: 'enterprise',
  chatgpt_enterprise: 'enterprise'
}

const GEMINI_TIER_ALIASES: Record<string, string> = {
  google_one_free: 'google_one_free',
  google_ai_pro: 'google_ai_pro',
  google_ai_ultra: 'google_ai_ultra',
  gcp_standard: 'gcp_standard',
  gcp_enterprise: 'gcp_enterprise',
  aistudio_free: 'aistudio_free',
  aistudio_paid: 'aistudio_paid',
  google_one_unknown: 'google_one_unknown',
  free: 'google_one_free',
  google_one_basic: 'google_one_free',
  google_one_standard: 'google_one_free',
  ai_premium: 'google_ai_pro',
  pro: 'google_ai_pro',
  ultra: 'google_ai_ultra'
}

function normalizeTierText(value: unknown): string {
  if (typeof value !== 'string') return ''
  return value.trim().toLowerCase().replace(/[\s-]+/g, '_')
}

function normalizeOpenAITier(value: unknown): string {
  const tier = normalizeTierText(value)
  return OPENAI_TIER_ALIASES[tier] || tier
}

function normalizeGeminiTier(value: unknown): string {
  const tier = normalizeTierText(value)
  return GEMINI_TIER_ALIASES[tier] || tier
}

function normalizeAntigravityPlanTier(value: unknown): string {
  const tier = normalizeTierText(value)
  if (tier === 'free' || tier === 'free_tier') return 'free-tier'
  if (tier === 'pro' || tier === 'g1_pro_tier') return 'g1-pro-tier'
  if (tier === 'ultra' || tier === 'g1_ultra_tier') return 'g1-ultra-tier'
  return ''
}

function getAntigravityTierFromRow(account: any): string | null {
  if (account.platform !== 'antigravity') return null
  const planTier = normalizeAntigravityPlanTier(account.credentials?.plan_type)
  if (planTier) return planTier
  const extra = account.extra as Record<string, unknown> | undefined
  if (!extra) return null
  const lca = extra.load_code_assist as Record<string, unknown> | undefined
  if (!lca) return null
  const paid = lca.paidTier as Record<string, unknown> | undefined
  if (paid && typeof paid.id === 'string') return paid.id
  const current = lca.currentTier as Record<string, unknown> | undefined
  if (current && typeof current.id === 'string') return current.id
  return null
}

interface ParsedTier {
  platform: string
  value: string
}

function parseSelectedTier(tier: string, fallbackPlatform: string): ParsedTier | null {
  const trimmed = String(tier || '').trim()
  if (!trimmed) return null
  const separator = trimmed.indexOf(':')
  if (separator >= 0) {
    return {
      platform: trimmed.slice(0, separator),
      value: trimmed.slice(separator + 1)
    }
  }
  return {
    platform: fallbackPlatform,
    value: trimmed
  }
}

export function matchesTier(
  account: Account,
  selectedTier: string,
  fallbackPlatform: string
): boolean {
  const tier = parseSelectedTier(selectedTier, fallbackPlatform)
  if (!tier || !tier.value) return true
  if (tier.platform && account.platform !== tier.platform) return false

  if (account.platform === 'openai') {
    return normalizeOpenAITier(account.credentials?.plan_type) === normalizeOpenAITier(tier.value)
  }
  if (account.platform === 'gemini') {
    return normalizeGeminiTier(account.credentials?.tier_id) === normalizeGeminiTier(tier.value)
  }
  if (account.platform === 'antigravity') {
    return getAntigravityTierFromRow(account) === tier.value
  }
  return false
}

// Export helper functions for UI display
export function getAntigravityTierFromAccount(account: any): string | null {
  return getAntigravityTierFromRow(account)
}
