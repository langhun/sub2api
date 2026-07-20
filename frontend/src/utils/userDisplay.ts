export interface UserDisplayIdentity {
  username?: string | null
  email?: string | null
}

export function userDisplayName(user: UserDisplayIdentity | null | undefined, fallback = ''): string {
  return user?.username?.trim() || user?.email?.trim() || fallback
}

export function userDisplayInitials(user: UserDisplayIdentity | null | undefined, fallback = 'U'): string {
  const label = userDisplayName(user)
  return label ? Array.from(label).slice(0, 2).join('').toUpperCase() : fallback
}
