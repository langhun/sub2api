export interface UsageUserIdentity {
  username?: string | null
  email?: string | null
}

export interface UsageUserDisplay {
  primary: string
  secondary: string
  title: string
}

/**
 * Custom presentation rule for the usage-log user cell: name first, email as
 * the stable secondary identifier.
 */
export function formatUsageUserDisplay(user?: UsageUserIdentity | null): UsageUserDisplay {
  const username = user?.username?.trim() || ''
  const email = user?.email?.trim() || ''
  const primary = username || email || '-'
  const secondary = username && email && username !== email ? email : ''

  return {
    primary,
    secondary,
    title: [username, email].filter(Boolean).join(' - ') || primary,
  }
}
