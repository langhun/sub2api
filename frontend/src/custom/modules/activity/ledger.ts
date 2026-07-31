const activityBalanceTypes = new Set(['checkin', 'checkin_luck', 'checkin_blindbox'])

export function isActivityBalanceType(type: string): boolean {
  return activityBalanceTypes.has(type)
}

export function activityHistoryTypeKey(type: string): string | undefined {
  return activityBalanceTypes.has(type) ? `redeem.activityHistoryTypes.${type}` : undefined
}
