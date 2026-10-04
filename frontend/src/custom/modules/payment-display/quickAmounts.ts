export const defaultQuickRechargeAmounts = [10, 20, 50, 100, 200, 500] as const

export function filterQuickRechargeAmounts(maxValue: unknown): number[] {
  const max = Number(maxValue)
  const limit = Number.isFinite(max) && max >= 0 ? max : 500
  return defaultQuickRechargeAmounts.filter((amount) => limit === 0 || amount <= limit)
}
