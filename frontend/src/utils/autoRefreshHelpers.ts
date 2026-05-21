export interface AutoRefreshSkipConditions {
  autoRefreshEnabled: boolean
  documentHidden: boolean
  loading: boolean
  autoRefreshFetching: boolean
  isAnyModalOpen: boolean
  menuShow: boolean
  showAccountToolsDropdown: boolean
  showAutoRefreshDropdown: boolean
  inSilentWindow: boolean
}

export function shouldSkipAutoRefresh(conditions: AutoRefreshSkipConditions): boolean {
  if (!conditions.autoRefreshEnabled) return true
  if (conditions.documentHidden) return true
  if (conditions.loading || conditions.autoRefreshFetching) return true
  if (conditions.isAnyModalOpen) return true
  if (conditions.menuShow || conditions.showAccountToolsDropdown || conditions.showAutoRefreshDropdown) return true
  if (conditions.inSilentWindow) return true

  return false
}

export function calculateSilentWindowCountdown(silentUntil: number, now: number): number {
  return Math.max(0, Math.ceil((silentUntil - now) / 1000))
}
