import type { ComposerTranslation } from 'vue-i18n'

const activityErrorKeys: Record<string, string> = {
  FEATURE_DISABLED: 'activityErrors.featureDisabled',
  INSUFFICIENT_BALANCE: 'activityErrors.insufficientBalance',
  LIMIT_EXCEEDED: 'activityErrors.limitExceeded',
  ALREADY_CLAIMED: 'activityErrors.alreadyClaimed',
  REDPACKET_EXPIRED: 'activityErrors.redpacketExpired',
  REDPACKET_EXHAUSTED: 'activityErrors.redpacketExhausted',
  REDPACKET_NOT_FOUND: 'activityErrors.redpacketNotFound',
  CANNOT_CLAIM_OWN_REDPACKET: 'activityErrors.cannotClaimOwn',
  IDEMPOTENCY_CONFLICT: 'activityErrors.duplicateRequest',
}

export function activityErrorMessage(error: unknown, t: ComposerTranslation, fallback: string): string {
  const data = (error as { response?: { data?: { code?: string; reason?: string; error?: string; message?: string; detail?: string } } })?.response?.data
  const code = data?.code || data?.reason
  const key = code ? activityErrorKeys[code] : undefined
  if (key) return t(key)
  return data?.detail || data?.error || data?.message || fallback
}
