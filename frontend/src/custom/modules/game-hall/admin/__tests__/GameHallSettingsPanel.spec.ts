import { createI18n } from 'vue-i18n'
import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import GameHallSettingsPanel from '../GameHallSettingsPanel.vue'

const api = vi.hoisted(() => ({
  getGameHallUserAccess: vi.fn(),
  updateGameHallUserAccess: vi.fn(),
}))

vi.mock('@/custom/modules/game-hall/api/admin', () => api)

const settings = {
  game_hall_enabled: true,
  game_slots_enabled: true,
  game_slots_min_bet: 0.01,
  game_slots_max_bet: 100,
  game_exchange_min_amount: 0.01,
  game_exchange_max_amount: 100,
  game_exchange_daily_limit: 1000,
  game_exchange_allow_dg_to_balance: true,
}

function mountPanel() {
  return mount(GameHallSettingsPanel, {
    props: { modelValue: settings },
    global: {
      plugins: [createI18n({ legacy: false, locale: 'en', messages: { en: {} } })],
    },
  })
}

describe('GameHallSettingsPanel user access control', () => {
  beforeEach(() => {
    api.getGameHallUserAccess.mockReset()
    api.updateGameHallUserAccess.mockReset()
  })

  it('loads a user access decision through the custom game-hall API', async () => {
    api.getGameHallUserAccess.mockResolvedValue({
      user_id: 7,
      disabled: false,
      updated_at: '2026-07-27T00:00:00Z',
    })
    const wrapper = mountPanel()

    await wrapper.get('[data-testid="game-hall-access-user-id"]').setValue('7')
    await wrapper.get('[data-testid="game-hall-access-load"]').trigger('click')
    await flushPromises()

    expect(api.getGameHallUserAccess).toHaveBeenCalledWith(7)
    expect(wrapper.get('[data-testid="game-hall-user-access"]').text()).toContain('User #7')
    expect(wrapper.get('[data-testid="game-hall-user-access"]').text()).toContain('Game hall is enabled')
  })

  it('updates the loaded decision without calling the core user API', async () => {
    api.getGameHallUserAccess.mockResolvedValue({
      user_id: 7,
      disabled: false,
      updated_at: '2026-07-27T00:00:00Z',
    })
    api.updateGameHallUserAccess.mockResolvedValue({
      user_id: 7,
      disabled: true,
      updated_at: '2026-07-27T00:01:00Z',
    })
    const wrapper = mountPanel()

    await wrapper.get('[data-testid="game-hall-access-user-id"]').setValue('7')
    await wrapper.get('[data-testid="game-hall-access-load"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="game-hall-user-access"] [role="switch"]').trigger('click')
    await flushPromises()

    expect(api.updateGameHallUserAccess).toHaveBeenCalledWith(7, true)
    expect(wrapper.get('[data-testid="game-hall-access-message"]').text()).toContain('Game hall disabled for this user')
  })

  it('rejects an invalid user ID before requesting the API', async () => {
    const wrapper = mountPanel()

    await wrapper.get('[data-testid="game-hall-access-load"]').trigger('click')

    expect(api.getGameHallUserAccess).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="game-hall-access-error"]').text()).toContain('Enter a valid user ID')
  })
})
