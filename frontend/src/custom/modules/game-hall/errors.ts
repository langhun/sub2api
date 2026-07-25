import type { ComposerTranslation } from 'vue-i18n'

const gameHallErrorKeys: Record<string, string> = {
  GAME_HALL_DISABLED: 'gameHall.errors.featureDisabled',
  GAME_HALL_USER_DISABLED: 'gameHall.errors.userDisabled',
  GAME_EXCHANGE_RETURN_DISABLED: 'gameHall.errors.exchangeReturnDisabled',
  GAME_EXCHANGE_DAILY_LIMIT: 'gameHall.errors.exchangeDailyLimit',
  GAME_EXCHANGE_OUT_OF_RANGE: 'gameHall.errors.exchangeOutOfRange',
}

export function gameHallErrorMessage(error: unknown, t: ComposerTranslation, fallback: string): string {
  type ErrorData = { code?: string; reason?: string; error?: string; message?: string; detail?: string }
  const typedError = error as ErrorData & { response?: { data?: ErrorData } }
  const data = typedError?.response?.data || typedError
  const code = data?.code || data?.reason
  const key = code ? gameHallErrorKeys[code] : undefined
  if (key) return t(key)
  return data?.detail || data?.error || data?.message || fallback
}
