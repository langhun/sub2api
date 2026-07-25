import { describe, expect, it, vi } from 'vitest'
import { activityErrorMessage } from '../errors'

describe('activityErrorMessage', () => {
  it('maps stable business codes to localized keys', () => {
    const t = vi.fn((key: string) => `translated:${key}`)
    expect(activityErrorMessage({ response: { data: { code: 'REDPACKET_EXPIRED' } } }, t, 'fallback'))
      .toBe('translated:activityErrors.redpacketExpired')
  })

  it('preserves a server detail when no stable code is known', () => {
    expect(activityErrorMessage({ response: { data: { detail: 'specific failure' } } }, vi.fn(), 'fallback'))
      .toBe('specific failure')
  })

  it('maps interceptor-normalized top-level activity errors', () => {
    const t = vi.fn((key: string) => `translated:${key}`)
    expect(activityErrorMessage({ status: 404, code: 'RECEIVER_NOT_FOUND', message: 'receiver not found' }, t, 'fallback'))
      .toBe('translated:activityErrors.receiverNotFound')
  })
})
