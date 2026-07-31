import { defineFeatureFlag } from '@/utils/featureFlags'
import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import GameHallSettingsPanel from './admin/GameHallSettingsPanel.vue'
export { gameHallPublicSettingsDefaults } from './publicSettings'

export const gameHallFeatureFlags = {
  gameHall: defineFeatureFlag({ key: 'game_hall_enabled', mode: 'opt-in', label: 'Game Hall' }),
} as const

interface GameHallSettings {
  game_hall_enabled: boolean
  game_slots_enabled: boolean
  game_slots_min_bet: number
  game_slots_max_bet: number
  game_exchange_min_amount: number
  game_exchange_max_amount: number
  game_exchange_daily_limit: number
  game_exchange_allow_dg_to_balance: boolean
}

const defaultGameHallSettings = (): GameHallSettings => ({
  game_hall_enabled: false,
  game_slots_enabled: false,
  game_slots_min_bet: 0.01,
  game_slots_max_bet: 1000,
  game_exchange_min_amount: 0.01,
  game_exchange_max_amount: 1000,
  game_exchange_daily_limit: 1000,
  game_exchange_allow_dg_to_balance: true,
})

function readSettings(settings: CustomSettingsValues): GameHallSettings {
  const defaults = defaultGameHallSettings()
  const values = settings as Record<string, unknown>
  const result = defaults as unknown as Record<string, unknown>
  for (const key of Object.keys(defaults) as Array<keyof GameHallSettings>) {
    const value = values[key]
    if (value !== null && value !== undefined) result[key] = value
  }
  return result as GameHallSettings
}

export const gameHallSettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'game-hall',
    placement: 'features',
    order: 10,
    component: GameHallSettingsPanel,
    settingKeys: Object.keys(defaultGameHallSettings()),
    createForm: () => defaultGameHallSettings(),
    fromSettings: readSettings,
    toPayload: (form) => {
      const settings = form as unknown as GameHallSettings
      return {
        game_hall_enabled: settings.game_hall_enabled,
        game_slots_enabled: settings.game_slots_enabled,
        game_slots_min_bet: Number(settings.game_slots_min_bet) || 0.01,
        game_slots_max_bet: Number(settings.game_slots_max_bet) || 0.01,
        game_exchange_min_amount: Number(settings.game_exchange_min_amount) || 0.01,
        game_exchange_max_amount: Number(settings.game_exchange_max_amount) || 0,
        game_exchange_daily_limit: Number(settings.game_exchange_daily_limit) || 0,
        game_exchange_allow_dg_to_balance: settings.game_exchange_allow_dg_to_balance,
      }
    },
    validate: (form) => {
      const settings = form as unknown as GameHallSettings
      const exchangeMin = Number(settings.game_exchange_min_amount)
      const exchangeMax = Number(settings.game_exchange_max_amount)
      const exchangeDaily = Number(settings.game_exchange_daily_limit)
      if (!Number.isFinite(exchangeMin) || exchangeMin <= 0 || !Number.isFinite(exchangeMax) || exchangeMax < 0 || !Number.isFinite(exchangeDaily) || exchangeDaily < 0) {
        return '兑换限额必须是有效数字，下限大于 0，上限与每日限额可为 0（不限）'
      }
      if (exchangeMax > 0 && exchangeMin > exchangeMax) return '单次兑换下限不能大于上限'
      if (settings.game_slots_enabled && Number(settings.game_slots_min_bet) > Number(settings.game_slots_max_bet)) return '最小投注不能大于最大投注'
      return ''
    },
  },
]
