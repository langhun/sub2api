import { describe, expect, it } from 'vitest'
import { gameHallNavigation } from '../navigation'
import { gameHallRoutes } from '../routes'

describe('game hall overlay registration', () => {
  it('preserves the authenticated feature-gated user route', () => {
    expect(gameHallRoutes).toContainEqual(expect.objectContaining({
      path: '/game-hall',
      name: 'GameHall',
      meta: expect.objectContaining({
        requiresAuth: true,
        requiresAdmin: false,
        requiresFeature: 'game_hall_enabled',
      }),
    }))
  })

  it('registers a feature-gated navigation item', () => {
    expect(gameHallNavigation).toContainEqual(expect.objectContaining({
      path: '/game-hall',
      labelKey: 'gameHall.title',
      hideInSimpleMode: true,
    }))
  })
})
