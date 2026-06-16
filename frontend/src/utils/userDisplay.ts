export interface UserDisplayLike {
  username?: string | null
  user_name?: string | null
  email?: string | null
  user_email?: string | null
}

function firstNonEmpty(...values: Array<string | null | undefined>): string {
  for (const value of values) {
    const trimmed = value?.trim()
    if (trimmed) {
      return trimmed
    }
  }
  return ''
}

export function getPreferredUserDisplayName(user?: UserDisplayLike | null, fallback = ''): string {
  return firstNonEmpty(user?.username, user?.user_name, user?.email, user?.user_email, fallback)
}

export function getSecondaryUserEmail(user?: UserDisplayLike | null): string {
  const username = firstNonEmpty(user?.username, user?.user_name)
  const email = firstNonEmpty(user?.email, user?.user_email)
  if (!username || !email || username === email) {
    return ''
  }
  return email
}

export function getUserDisplayInitial(user?: UserDisplayLike | null, fallback = '?'): string {
  const displayName = getPreferredUserDisplayName(user)
  return displayName.charAt(0).toUpperCase() || fallback
}
