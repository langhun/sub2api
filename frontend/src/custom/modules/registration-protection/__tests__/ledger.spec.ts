import { describe, expect, it } from 'vitest'
import { registrationHistoryTypeKey, isRegistrationBalanceType } from '../ledger'

describe('registration balance ledger display', () => {
  it('renders signup bonuses as a balance history type', () => {
    expect(isRegistrationBalanceType('signup_bonus')).toBe(true)
    expect(registrationHistoryTypeKey('signup_bonus')).toBe('redeem.signupBonus')
  })

  it('does not classify unrelated history types as signup bonuses', () => {
    expect(isRegistrationBalanceType('balance')).toBe(false)
    expect(registrationHistoryTypeKey('checkin')).toBeUndefined()
  })
})
