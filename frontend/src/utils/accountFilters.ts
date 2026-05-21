/**
 * @fileoverview 账户过滤工具函数
 *
 * 提供多维度的账户过滤功能，包括平台、类型、状态、套餐等级、分组、隐私模式和搜索。
 *
 * @example
 * ```typescript
 * import { matchesPlatform, matchesTier, matchesSearch } from '@/utils/accountFilters'
 *
 * const filtered = accounts.filter(account => {
 *   return matchesPlatform(account, 'openai') &&
 *          matchesTier(account, 'plus', 'openai') &&
 *          matchesSearch(account, searchTerm)
 * })
 * ```
 */

import type { Account } from '@/types'
import {
  matchesAccountMainStatusFilter,
  matchesAccountRuntimeStatusFilter,
  matchesAccountSchedulingStatusFilter,
  type AccountMainStatusFilterValue,
  type AccountRuntimeStatusFilterValue,
  type AccountSchedulingStatusFilterValue
} from './accountStatus'

/** 未分组账户的特殊查询值 */
export const ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE = 'ungrouped'

/** 隐私模式未设置的特殊查询值 */
export const ACCOUNT_PRIVACY_MODE_UNSET_QUERY_VALUE = '__unset__'

/**
 * 账户过滤选项
 */
/**
 * 账户过滤选项
 */
export interface AccountFilterOptions {
  /** 平台过滤 */
  platform?: string
  /** 套餐等级过滤 */
  tier?: string
  /** 账户类型过滤 */
  type?: string
  /** 主状态过滤 */
  main_status?: string
  /** 运行时状态过滤 */
  runtime_status?: string
  /** 调度状态过滤 */
  scheduling_status?: string
  /** 隐私模式过滤 */
  privacy_mode?: string
  /** 分组过滤 */
  group?: string
  /** 搜索关键词 */
  search?: string
}

/**
 * 状态评估选项
 */
export interface StatusEvalOptions {
  /** 当前时间戳（毫秒） */
  nowMs: number
}

/**
 * 匹配账户平台
 *
 * @param account 账户对象
 * @param platform 平台名称，空字符串匹配所有
 * @returns 是否匹配
 */
export function matchesPlatform(account: Account, platform: string): boolean {
  if (!platform) return true
  return account.platform === platform
}

/**
 * 匹配账户类型
 *
 * @param account 账户对象
 * @param type 账户类型，空字符串匹配所有
 * @returns 是否匹配
 */
export function matchesType(account: Account, type: string): boolean {
  if (!type) return true
  return account.type === type
}

/**
 * 匹配账户状态（主状态、运行时状态、调度状态）
 *
 * @param account 账户对象
 * @param mainStatus 主状态
 * @param runtimeStatus 运行时状态
 * @param schedulingStatus 调度状态
 * @param options 状态评估选项（包含当前时间戳）
 * @returns 是否匹配
 */
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

/**
 * 匹配账户分组
 *
 * @param account 账户对象
 * @param group 分组 ID 或特殊值 'ungrouped'
 * @returns 是否匹配
 */
export function matchesGroup(account: Account, group: string): boolean {
  if (!group) return true

  const groupIds = account.group_ids ?? account.groups?.map((g) => g.id) ?? []

  if (group === ACCOUNT_UNGROUPED_GROUP_QUERY_VALUE) {
    return groupIds.length === 0
  }

  return groupIds.includes(Number(group))
}

/**
 * 匹配账户隐私模式
 *
 * @param account 账户对象
 * @param privacyMode 隐私模式或特殊值 '__unset__'
 * @returns 是否匹配
 */
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

/**
 * 匹配搜索关键词（在账户名称中搜索）
 *
 * @param account 账户对象
 * @param search 搜索关键词，不区分大小写
 * @returns 是否匹配
 */
export function matchesSearch(account: Account, search: string): boolean {
  const searchTerm = String(search || '').trim().toLowerCase()
  if (!searchTerm) return true

  return account.name.toLowerCase().includes(searchTerm)
}

/** OpenAI 套餐等级别名映射 */
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

/** Gemini 套餐等级别名映射 */
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

/**
 * 标准化套餐等级文本（转小写，替换空格和连字符为下划线）
 */
function normalizeTierText(value: unknown): string {
  if (typeof value !== 'string') return ''
  return value.trim().toLowerCase().replace(/[\s-]+/g, '_')
}

/**
 * 标准化 OpenAI 套餐等级（处理别名）
 */
function normalizeOpenAITier(value: unknown): string {
  const tier = normalizeTierText(value)
  return OPENAI_TIER_ALIASES[tier] || tier
}

/**
 * 标准化 Gemini 套餐等级（处理别名）
 */
function normalizeGeminiTier(value: unknown): string {
  const tier = normalizeTierText(value)
  return GEMINI_TIER_ALIASES[tier] || tier
}

/**
 * 标准化 Antigravity 套餐等级
 */
function normalizeAntigravityPlanTier(value: unknown): string {
  const tier = normalizeTierText(value)
  if (tier === 'free' || tier === 'free_tier') return 'free-tier'
  if (tier === 'pro' || tier === 'g1_pro_tier') return 'g1-pro-tier'
  if (tier === 'ultra' || tier === 'g1_ultra_tier') return 'g1-ultra-tier'
  return ''
}

/**
 * 从 Antigravity 账户获取套餐等级
 * @internal
 */
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

/**
 * 解析的套餐等级信息
 * @internal
 */
interface ParsedTier {
  platform: string
  value: string
}

/**
 * 解析选中的套餐等级（支持 'platform:tier' 格式）
 * @internal
 */
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

/**
 * 匹配账户套餐等级
 *
 * 支持平台前缀格式（如 'openai:plus'）和别名（如 'chatgpt_plus' -> 'plus'）
 *
 * @param account 账户对象
 * @param selectedTier 选中的套餐等级，可以是 'tier' 或 'platform:tier' 格式
 * @param fallbackPlatform 当 selectedTier 不包含平台前缀时使用的默认平台
 * @returns 是否匹配
 *
 * @example
 * ```typescript
 * matchesTier(account, 'plus', 'openai')           // 基本用法
 * matchesTier(account, 'openai:plus', '')          // 带平台前缀
 * matchesTier(account, 'chatgpt_plus', 'openai')  // 使用别名
 * ```
 */
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

/**
 * 获取 Antigravity 账户的套餐等级
 *
 * @param account 账户对象
 * @returns 套餐等级 ID，如果无法确定则返回 null
 */
export function getAntigravityTierFromAccount(account: any): string | null {
  return getAntigravityTierFromRow(account)
}
