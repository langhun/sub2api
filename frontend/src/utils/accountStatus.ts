import type { Account } from '@/types'

export type ParsedTempUnschedReason = {
  statusCode: number | null
  matchedKeyword: string | null
  errorMessage: string | null
  ruleIndex: number | null
}

export type AccountStatusFilterValue =
  | ''
  | 'active'
  | 'inactive'
  | 'error'
  | 'rate_limited'
  | 'overloaded'
  | 'temp_unschedulable'
  | 'unschedulable'

export type StatusTone = 'success' | 'warning' | 'danger' | 'gray'

export type AccountMainStatusCode =
  | 'main_active'
  | 'main_inactive'
  | 'main_error'

export type AccountSchedulingStatusCode =
  | 'schedule_enabled'
  | 'schedule_manual_paused'
  | 'schedule_expired_paused'

export type AccountRuntimeStatusCode =
  | 'runtime_normal'
  | 'runtime_rate_limited'
  | 'runtime_overloaded'
  | 'runtime_oauth401_cooldown'
  | 'runtime_forbidden_cooldown'
  | 'runtime_http_cooldown'
  | 'runtime_stream_timeout_cooldown'
  | 'runtime_token_refresh_cooldown'
  | 'runtime_quota_exceeded'
  | 'runtime_temp_unschedulable'

export type AccountStatusBadgeState<TCode extends string> = {
  code: TCode
  tone: StatusTone
}

export type AccountRuntimeStatusState = AccountStatusBadgeState<AccountRuntimeStatusCode> & {
  clickable: boolean
  until: string | null
  parsedReason: ParsedTempUnschedReason
  statusCode: number | null
}

const tempUnschedStatusCodePattern = /\b([1-5][0-9]{2})\b/

export function isFutureAt(value: string | null | undefined): boolean {
  if (!value) return false
  const time = new Date(value).getTime()
  return Number.isFinite(time) && time > Date.now()
}

export function isExpiredAutoPaused(account: Pick<Account, 'auto_pause_on_expired' | 'expires_at'>): boolean {
  if (!account.auto_pause_on_expired || !account.expires_at) return false
  return account.expires_at * 1000 <= Date.now()
}

export function getBadgeClassForTone(tone: StatusTone): string {
  switch (tone) {
    case 'success':
      return 'badge-success'
    case 'warning':
      return 'badge-warning'
    case 'danger':
      return 'badge-danger'
    default:
      return 'badge-gray'
  }
}

export function parseTempUnschedReason(reason: string | null | undefined): ParsedTempUnschedReason {
  const trimmed = reason?.trim()
  if (!trimmed) {
    return {
      statusCode: null,
      matchedKeyword: null,
      errorMessage: null,
      ruleIndex: null
    }
  }

  try {
    const parsed = JSON.parse(trimmed) as Partial<{
      status_code: number
      matched_keyword: string
      error_message: string
      rule_index: number
    }>
    return {
      statusCode: Number.isFinite(parsed.status_code) ? Number(parsed.status_code) : null,
      matchedKeyword: typeof parsed.matched_keyword === 'string' && parsed.matched_keyword
        ? parsed.matched_keyword
        : null,
      errorMessage: typeof parsed.error_message === 'string' && parsed.error_message
        ? parsed.error_message
        : trimmed,
      ruleIndex: Number.isFinite(parsed.rule_index) ? Number(parsed.rule_index) : null
    }
  } catch {
    const statusCodeMatch = trimmed.match(tempUnschedStatusCodePattern)
    const lower = trimmed.toLowerCase()
    let matchedKeyword: string | null = null

    if (lower.includes('stream_timeout') || lower.includes('stream data interval timeout')) {
      matchedKeyword = 'stream_timeout'
    } else if (lower.includes('token refresh retry exhausted')) {
      matchedKeyword = 'token_refresh'
    } else if (lower.includes('internal 500')) {
      matchedKeyword = 'internal_500'
    } else if (lower.includes('oauth 401')) {
      matchedKeyword = 'oauth_401'
    } else if (lower.includes('temporary cooldown')) {
      matchedKeyword = 'temporary_cooldown'
    }

    return {
      statusCode: statusCodeMatch ? Number(statusCodeMatch[1]) : null,
      matchedKeyword,
      errorMessage: trimmed,
      ruleIndex: -1
    }
  }
}

function isQuotaExceeded(account: Pick<
  Account,
  'quota_limit' | 'quota_used' | 'quota_daily_limit' | 'quota_daily_used' | 'quota_weekly_limit' | 'quota_weekly_used'
>): boolean {
  const exceeded = (used?: number | null, limit?: number | null) =>
    typeof limit === 'number' && limit > 0 && typeof used === 'number' && used >= limit

  return (
    exceeded(account.quota_used, account.quota_limit) ||
    exceeded(account.quota_daily_used, account.quota_daily_limit) ||
    exceeded(account.quota_weekly_used, account.quota_weekly_limit)
  )
}

export function isAccountRateLimited(account: Pick<Account, 'rate_limit_reset_at'>): boolean {
  return isFutureAt(account.rate_limit_reset_at)
}

export function isAccountOverloaded(account: Pick<Account, 'overload_until'>): boolean {
  return isFutureAt(account.overload_until)
}

export function isAccountTempUnschedulable(account: Pick<Account, 'temp_unschedulable_until'>): boolean {
  return isFutureAt(account.temp_unschedulable_until)
}

export function isAccountInactive(account: Pick<Account, 'status'>): boolean {
  return account.status !== 'active' && account.status !== 'error'
}

export function isTempUnschedRuntimeCode(code: AccountRuntimeStatusCode): boolean {
  return (
    code === 'runtime_oauth401_cooldown' ||
    code === 'runtime_forbidden_cooldown' ||
    code === 'runtime_http_cooldown' ||
    code === 'runtime_stream_timeout_cooldown' ||
    code === 'runtime_token_refresh_cooldown' ||
    code === 'runtime_temp_unschedulable'
  )
}

function hasSchedulingBlockingRuntime(code: AccountRuntimeStatusCode): boolean {
  return code === 'runtime_rate_limited' || code === 'runtime_overloaded' || isTempUnschedRuntimeCode(code)
}

export function matchesAccountStatusFilter(
  account: Pick<
    Account,
    | 'status'
    | 'schedulable'
    | 'auto_pause_on_expired'
    | 'expires_at'
    | 'rate_limit_reset_at'
    | 'overload_until'
    | 'temp_unschedulable_until'
    | 'temp_unschedulable_reason'
    | 'quota_limit'
    | 'quota_used'
    | 'quota_daily_limit'
    | 'quota_daily_used'
    | 'quota_weekly_limit'
    | 'quota_weekly_used'
  >,
  filter: AccountStatusFilterValue
): boolean {
  const mainState = getAccountMainStatusState(account)
  const schedulingState = getAccountSchedulingStatusState(account)
  const runtimeState = getAccountRuntimeStatusState(account)

  switch (filter) {
    case '':
      return true
    case 'active':
      return (
        mainState.code === 'main_active' &&
        schedulingState.code === 'schedule_enabled' &&
        !hasSchedulingBlockingRuntime(runtimeState.code)
      )
    case 'inactive':
      return mainState.code === 'main_inactive'
    case 'error':
      return mainState.code === 'main_error'
    case 'rate_limited':
      return mainState.code === 'main_active' && runtimeState.code === 'runtime_rate_limited'
    case 'overloaded':
      return mainState.code === 'main_active' && runtimeState.code === 'runtime_overloaded'
    case 'temp_unschedulable':
      return mainState.code === 'main_active' && isTempUnschedRuntimeCode(runtimeState.code)
    case 'unschedulable':
      return (
        mainState.code === 'main_active' &&
        schedulingState.code !== 'schedule_enabled' &&
        !hasSchedulingBlockingRuntime(runtimeState.code)
      )
    default:
      return true
  }
}

export function getAccountMainStatusState(
  account: Pick<Account, 'status'>
): AccountStatusBadgeState<AccountMainStatusCode> {
  if (account.status === 'error') {
    return { code: 'main_error', tone: 'danger' }
  }
  if (account.status !== 'active') {
    return { code: 'main_inactive', tone: 'gray' }
  }
  return { code: 'main_active', tone: 'success' }
}

export function getAccountSchedulingStatusState(
  account: Pick<Account, 'schedulable' | 'auto_pause_on_expired' | 'expires_at'>
): AccountStatusBadgeState<AccountSchedulingStatusCode> {
  if (isExpiredAutoPaused(account)) {
    return { code: 'schedule_expired_paused', tone: 'gray' }
  }
  if (!account.schedulable) {
    return { code: 'schedule_manual_paused', tone: 'gray' }
  }
  return { code: 'schedule_enabled', tone: 'success' }
}

export function getTempUnschedRuntimeCode(
  parsedReason: ParsedTempUnschedReason
): AccountRuntimeStatusCode {
  if (parsedReason.statusCode === 401 || parsedReason.matchedKeyword === 'oauth_401') {
    return 'runtime_oauth401_cooldown'
  }
  if (parsedReason.statusCode === 403) {
    return 'runtime_forbidden_cooldown'
  }
  if (parsedReason.matchedKeyword === 'stream_timeout') {
    return 'runtime_stream_timeout_cooldown'
  }
  if (parsedReason.matchedKeyword === 'token_refresh') {
    return 'runtime_token_refresh_cooldown'
  }
  if (parsedReason.statusCode) {
    return 'runtime_http_cooldown'
  }
  return 'runtime_temp_unschedulable'
}

export function getAccountRuntimeStatusState(
  account: Pick<
    Account,
    | 'rate_limit_reset_at'
    | 'overload_until'
    | 'temp_unschedulable_until'
    | 'temp_unschedulable_reason'
    | 'quota_limit'
    | 'quota_used'
    | 'quota_daily_limit'
    | 'quota_daily_used'
    | 'quota_weekly_limit'
    | 'quota_weekly_used'
  >
): AccountRuntimeStatusState {
  const parsedReason = parseTempUnschedReason(account.temp_unschedulable_reason)

  if (isAccountRateLimited(account)) {
    return {
      code: 'runtime_rate_limited',
      tone: 'warning',
      clickable: false,
      until: account.rate_limit_reset_at,
      parsedReason,
      statusCode: null
    }
  }

  if (isAccountOverloaded(account)) {
    return {
      code: 'runtime_overloaded',
      tone: 'danger',
      clickable: false,
      until: account.overload_until,
      parsedReason,
      statusCode: null
    }
  }

  if (isAccountTempUnschedulable(account)) {
    return {
      code: getTempUnschedRuntimeCode(parsedReason),
      tone: 'warning',
      clickable: true,
      until: account.temp_unschedulable_until,
      parsedReason,
      statusCode: parsedReason.statusCode
    }
  }

  if (isQuotaExceeded(account)) {
    return {
      code: 'runtime_quota_exceeded',
      tone: 'warning',
      clickable: false,
      until: null,
      parsedReason,
      statusCode: null
    }
  }

  return {
    code: 'runtime_normal',
    tone: 'success',
    clickable: false,
    until: null,
    parsedReason,
    statusCode: null
  }
}
