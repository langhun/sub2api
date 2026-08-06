export interface UsageUserIdentity {
  username?: string | null
  email?: string | null
}

export interface UsageUserDisplay {
  label: string
  title: string
}

/**
 * Custom presentation rule for the usage-log user cell: name first, email as
 * the stable secondary identifier.
 */
export function formatUsageUserDisplay(user?: UsageUserIdentity | null): UsageUserDisplay {
  const username = user?.username?.trim() || ''
  const email = user?.email?.trim() || ''
  const label = username || email || '-'

  return {
    label,
    title: [username, email].filter(Boolean).join(' - ') || label,
  }
}
