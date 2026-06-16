import { describe, expect, it } from 'vitest'

import {
  getPreferredUserDisplayName,
  getSecondaryUserEmail,
  getUserDisplayInitial,
} from '../userDisplay'

describe('userDisplay', () => {
  it('prefers username and keeps email as secondary text', () => {
    const user = {
      username: 'Alice',
      email: 'alice@example.com',
    }

    expect(getPreferredUserDisplayName(user)).toBe('Alice')
    expect(getSecondaryUserEmail(user)).toBe('alice@example.com')
    expect(getUserDisplayInitial(user)).toBe('A')
  })

  it('falls back to email when username is empty', () => {
    const user = {
      username: '  ',
      email: 'fallback@example.com',
    }

    expect(getPreferredUserDisplayName(user)).toBe('fallback@example.com')
    expect(getSecondaryUserEmail(user)).toBe('')
    expect(getUserDisplayInitial(user)).toBe('F')
  })

  it('supports legacy user_name and user_email fields', () => {
    const user = {
      user_name: 'Legacy Name',
      user_email: 'legacy@example.com',
    }

    expect(getPreferredUserDisplayName(user)).toBe('Legacy Name')
    expect(getSecondaryUserEmail(user)).toBe('legacy@example.com')
    expect(getUserDisplayInitial(user)).toBe('L')
  })
})
