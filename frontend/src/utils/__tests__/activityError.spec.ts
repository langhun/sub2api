import { describe, expect, it, vi } from 'vitest'
import { activityErrorMessage } from '@/utils/activityError'

describe('activityErrorMessage', () => {
  it('translates per-user game hall governance errors', () => {
    const t = (key: string) => `translated:${key}`
    expect(activityErrorMessage({ reason: 'GAME_HALL_USER_DISABLED' }, t as never, 'fallback'))
      .toBe('translated:activityErrors.gameHallUserDisabled')
  })
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
