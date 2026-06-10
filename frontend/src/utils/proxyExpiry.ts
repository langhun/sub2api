type ProxyExpiryStatus = 'active' | 'inactive' | 'expired' | string | undefined

interface ProxyExpiryLabel {
  key: string
  params?: { days: number }
}

const MS_PER_DAY = 24 * 60 * 60 * 1000
const DEFAULT_WARN_DAYS = 7

function startOfDayMs(value: Date): number {
  return new Date(value.getFullYear(), value.getMonth(), value.getDate()).getTime()
}

function getExpiryDiffDays(expiresAt?: string | null): number | null {
  if (!expiresAt) return null

  const parsed = new Date(expiresAt)
  if (Number.isNaN(parsed.getTime())) return null

  const todayMs = startOfDayMs(new Date())
  const expiryMs = startOfDayMs(parsed)
  return Math.ceil((expiryMs - todayMs) / MS_PER_DAY)
}

export function proxyExpiryLabelKey(
  expiresAt?: string | null,
  status?: ProxyExpiryStatus
): ProxyExpiryLabel {
  const diffDays = getExpiryDiffDays(expiresAt)

  if (!expiresAt || diffDays === null) {
    return { key: 'admin.proxies.neverExpires' }
  }

  if (status === 'expired' || diffDays < 0) {
    if (diffDays <= -1) {
      return { key: 'admin.proxies.overdueDays', params: { days: Math.abs(diffDays) } }
    }
    return { key: 'admin.proxies.expired' }
  }

  if (diffDays === 0) {
    return { key: 'admin.proxies.expiringInDays', params: { days: 0 } }
  }

  if (diffDays <= DEFAULT_WARN_DAYS) {
    return { key: 'admin.proxies.expiringInDays', params: { days: diffDays } }
  }

  return { key: 'admin.proxies.remainingDays', params: { days: diffDays } }
}

export function proxyExpiryBadgeClass(
  expiresAt?: string | null,
  status?: ProxyExpiryStatus
): string {
  const diffDays = getExpiryDiffDays(expiresAt)

  if (!expiresAt || diffDays === null) {
    return 'badge badge-gray'
  }

  if (status === 'expired' || diffDays < 0) {
    return 'badge badge-danger'
  }

  if (diffDays <= DEFAULT_WARN_DAYS) {
    return 'badge badge-warning'
  }

  return 'badge badge-success'
}
