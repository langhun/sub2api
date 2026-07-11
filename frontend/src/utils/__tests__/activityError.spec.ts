import { describe, expect, it, vi } from 'vitest'
import { activityErrorMessage } from '@/utils/activityError'

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
})
