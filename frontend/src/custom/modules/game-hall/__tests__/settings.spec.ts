import { describe, expect, it } from 'vitest'
import { gameHallSettingsPanels } from '../settings'

describe('game hall settings panel', () => {
  it('keeps values returned by the admin settings API', () => {
    const form = gameHallSettingsPanels[0].fromSettings({
      game_hall_enabled: true,
      game_slots_max_bet: 42,
    }) as Record<string, unknown>

    expect(form.game_hall_enabled).toBe(true)
    expect(form.game_slots_max_bet).toBe(42)
  })
})
