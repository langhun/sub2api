import { describe, expect, it, vi } from 'vitest'
import { gameHallErrorMessage } from '../errors'

describe('game hall error messages', () => {
  it('translates game-hall-specific server errors', () => {
    const t = (key: string) => `translated:${key}`

    expect(gameHallErrorMessage({ reason: 'GAME_HALL_USER_DISABLED' }, t as never, 'fallback'))
      .toBe('translated:gameHall.errors.userDisabled')
  })

  it('preserves a server detail when the error is not a stable game-hall code', () => {
    expect(gameHallErrorMessage({ response: { data: { detail: 'specific failure' } } }, vi.fn(), 'fallback'))
      .toBe('specific failure')
  })
})
