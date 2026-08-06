import { describe, expect, it } from 'vitest'
import { formatUsageUserDisplay } from '../usage'

describe('usage user display', () => {
  it('shows the username before the email', () => {
    expect(formatUsageUserDisplay({ username: 'Alice', email: 'alice@example.com' })).toEqual({
      label: 'Alice',
      title: 'Alice - alice@example.com',
    })
  })

  it('falls back to the email when a username is absent', () => {
    expect(formatUsageUserDisplay({ username: '  ', email: 'alice@example.com' })).toEqual({
      label: 'alice@example.com',
      title: 'alice@example.com',
    })
  })
})
