import { describe, expect, it } from 'vitest'
import { userDisplayInitials, userDisplayName } from '@/utils/userDisplay'

describe('user display identity', () => {
  it('prefers a non-empty username', () => {
    expect(userDisplayName({ username: '  Alice Chen  ', email: 'alice@example.com' })).toBe('Alice Chen')
    expect(userDisplayInitials({ username: 'Alice', email: 'alice@example.com' })).toBe('AL')
  })

  it('falls back to the complete email and then the supplied fallback', () => {
    expect(userDisplayName({ username: '  ', email: '  alice@example.com  ' })).toBe('alice@example.com')
    expect(userDisplayName(null, 'User #7')).toBe('User #7')
  })

  it('supports unicode initials without splitting surrogate pairs', () => {
    expect(userDisplayInitials({ username: '😀用户' })).toBe('😀用')
  })
})
