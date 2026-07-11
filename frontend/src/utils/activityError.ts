import type { ComposerTranslation } from 'vue-i18n'

const activityErrorKeys: Record<string, string> = {
  FEATURE_DISABLED: 'activityErrors.featureDisabled',
  TRANSFER_DISABLED: 'activityErrors.featureDisabled',
  REDPACKET_DISABLED: 'activityErrors.featureDisabled',
  GAME_HALL_DISABLED: 'activityErrors.featureDisabled',
  GAME_HALL_USER_DISABLED: 'activityErrors.gameHallUserDisabled',
  GAME_EXCHANGE_RETURN_DISABLED: 'activityErrors.gameExchangeReturnDisabled',
  GAME_EXCHANGE_DAILY_LIMIT: 'activityErrors.gameExchangeDailyLimit',
  GAME_EXCHANGE_OUT_OF_RANGE: 'activityErrors.gameExchangeOutOfRange',
  INSUFFICIENT_BALANCE: 'activityErrors.insufficientBalance',
  TRANSFER_INSUFFICIENT: 'activityErrors.insufficientBalance',
  LIMIT_EXCEEDED: 'activityErrors.limitExceeded',
  TRANSFER_DAILY_LIMIT: 'activityErrors.transferDailyLimit',
  TRANSFER_DAILY_COUNT: 'activityErrors.transferDailyCount',
  TRANSFER_SELF: 'activityErrors.transferSelf',
  TRANSFER_AMOUNT_INVALID: 'activityErrors.transferAmountInvalid',
  RECEIVER_NOT_FOUND: 'activityErrors.receiverNotFound',
  RECEIVER_QUERY_INVALID: 'activityErrors.receiverNotFound',
  RECEIVER_AMBIGUOUS: 'activityErrors.receiverNotFound',
  ALREADY_CLAIMED: 'activityErrors.alreadyClaimed',
  REDPACKET_ALREADY_CLAIMED: 'activityErrors.alreadyClaimed',
  REDPACKET_EXPIRED: 'activityErrors.redpacketExpired',
  REDPACKET_EXHAUSTED: 'activityErrors.redpacketExhausted',
  REDPACKET_NOT_FOUND: 'activityErrors.redpacketNotFound',
  CANNOT_CLAIM_OWN_REDPACKET: 'activityErrors.cannotClaimOwn',
  REDPACKET_SELF_CLAIM: 'activityErrors.cannotClaimOwn',
  IDEMPOTENCY_CONFLICT: 'activityErrors.duplicateRequest',
}

export function activityErrorMessage(error: unknown, t: ComposerTranslation, fallback: string): string {
  type ErrorData = { code?: string; reason?: string; error?: string; message?: string; detail?: string }
  const typedError = error as ErrorData & { response?: { data?: ErrorData } }
  const data = typedError?.response?.data || typedError
  const code = data?.code || data?.reason
  const key = code ? activityErrorKeys[code] : undefined
  if (key) return t(key)
  return data?.detail || data?.error || data?.message || fallback
}
