import { defineFeatureFlag } from '@/utils/featureFlags'
import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import ActivityEntrySwitchesPanel from './admin/ActivityEntrySwitchesPanel.vue'
import ActivitySettingsPanel from './admin/ActivitySettingsPanel.vue'
export { activityPublicSettingsDefaults } from './publicSettings'

export type CodeCharacterSet = 'uppercase' | 'numeric' | 'alphanumeric'

export interface CodeFormatRule {
  prefix: string
  character_set: CodeCharacterSet
  separator: string
  group_length: number
  group_count: number
}

export type CodeFormatSettings = Record<
  'balance' | 'concurrency' | 'subscription' | 'invitation' | 'redpacket',
  CodeFormatRule
>

export const activityFeatureFlags = {
  usageQuery: defineFeatureFlag({ key: 'usage_query_enabled', mode: 'opt-out', label: 'Usage Query' }),
  checkin: defineFeatureFlag({ key: 'checkin_enabled', mode: 'opt-in', label: 'Check-in' }),
  checkinLuck: defineFeatureFlag({ key: 'checkin_luck_enabled', mode: 'opt-in', label: 'Lucky Check-in' }),
  checkinBlindbox: defineFeatureFlag({ key: 'checkin_blindbox_enabled', mode: 'opt-in', label: 'Check-in Blind Box' }),
  leaderboard: defineFeatureFlag({ key: 'leaderboard_enabled', mode: 'opt-out', label: 'Leaderboard' }),
  leaderboardBalance: defineFeatureFlag({ key: 'leaderboard_balance_enabled', mode: 'opt-out', label: 'Balance Leaderboard' }),
  leaderboardConsumption: defineFeatureFlag({ key: 'leaderboard_consumption_enabled', mode: 'opt-out', label: 'Consumption Leaderboard' }),
  leaderboardCheckin: defineFeatureFlag({ key: 'leaderboard_checkin_enabled', mode: 'opt-out', label: 'Check-in Leaderboard' }),
  leaderboardTransfer: defineFeatureFlag({ key: 'leaderboard_transfer_enabled', mode: 'opt-in', label: 'Transfer Leaderboard' }),
} as const

interface ActivityLeaderboardSettings {
  leaderboard_enabled: boolean
  leaderboard_balance_enabled: boolean
  leaderboard_consumption_enabled: boolean
  leaderboard_checkin_enabled: boolean
  leaderboard_transfer_enabled: boolean
  leaderboard_include_admin: boolean
}

interface ActivityUsageSettings {
  usage_query_enabled: boolean
}

type ActivityEntrySwitchesSettings = ActivityUsageSettings & ActivityLeaderboardSettings

interface ActivitySettings {
  checkin_enabled: boolean
  checkin_min_balance: number
  checkin_max_balance: number
  checkin_luck_enabled: boolean
  checkin_luck_min_multiplier: number
  checkin_luck_max_multiplier: number
  checkin_blindbox_enabled: boolean
  checkin_blindbox_trigger_type: string
  checkin_blindbox_interval: number
  code_format_settings: CodeFormatSettings
  code_format_settings_valid: boolean
}

const defaultCodeFormatSettings = (): CodeFormatSettings => ({
  balance: { prefix: 'BAL', character_set: 'alphanumeric', separator: '-', group_length: 4, group_count: 3 },
  concurrency: { prefix: 'CON', character_set: 'alphanumeric', separator: '-', group_length: 4, group_count: 3 },
  subscription: { prefix: 'SUB', character_set: 'alphanumeric', separator: '-', group_length: 4, group_count: 3 },
  invitation: { prefix: 'INV', character_set: 'alphanumeric', separator: '-', group_length: 4, group_count: 3 },
  redpacket: { prefix: 'RP', character_set: 'alphanumeric', separator: '-', group_length: 4, group_count: 3 },
})

const defaultLeaderboardSettings = (): ActivityLeaderboardSettings => ({
  leaderboard_enabled: true,
  leaderboard_balance_enabled: true,
  leaderboard_consumption_enabled: true,
  leaderboard_checkin_enabled: true,
  leaderboard_transfer_enabled: false,
  leaderboard_include_admin: false,
})

const defaultActivityUsageSettings = (): ActivityUsageSettings => ({
  usage_query_enabled: true,
})

const defaultActivityEntrySwitchesSettings = (): ActivityEntrySwitchesSettings => ({
  ...defaultActivityUsageSettings(),
  ...defaultLeaderboardSettings(),
})

const defaultActivitySettings = (): ActivitySettings => ({
  checkin_enabled: false,
  checkin_min_balance: 0.1,
  checkin_max_balance: 1,
  checkin_luck_enabled: false,
  checkin_luck_min_multiplier: 0.1,
  checkin_luck_max_multiplier: 3,
  checkin_blindbox_enabled: false,
  checkin_blindbox_trigger_type: 'streak',
  checkin_blindbox_interval: 7,
  code_format_settings: defaultCodeFormatSettings(),
  code_format_settings_valid: true,
})

function readSettings<T extends object>(defaults: T, settings: CustomSettingsValues): T {
  const result = { ...defaults }
  const target = result as Record<string, unknown>
  const values = settings as Record<string, unknown>
  for (const key of Object.keys(defaults) as Array<keyof T>) {
    const value = values[key as string]
    if (value !== null && value !== undefined) {
      target[key as string] = value
    }
  }
  return result
}

export const activitySettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'activity-entry-switches',
    placement: 'entry-switches',
    order: 10,
    component: ActivityEntrySwitchesPanel,
    settingKeys: Object.keys(defaultActivityEntrySwitchesSettings()),
    createForm: defaultActivityEntrySwitchesSettings,
    fromSettings: (settings) => readSettings(defaultActivityEntrySwitchesSettings(), settings),
    toPayload: (form) => {
      const settings = form as unknown as ActivityEntrySwitchesSettings
      return {
        usage_query_enabled: settings.usage_query_enabled,
        leaderboard_enabled: settings.leaderboard_enabled,
        leaderboard_balance_enabled: settings.leaderboard_balance_enabled,
        leaderboard_consumption_enabled: settings.leaderboard_consumption_enabled,
        leaderboard_checkin_enabled: settings.leaderboard_checkin_enabled,
        leaderboard_transfer_enabled: settings.leaderboard_transfer_enabled,
        leaderboard_include_admin: settings.leaderboard_include_admin,
      }
    },
    validate: (form, allForms) => {
      const settings = form as unknown as ActivityEntrySwitchesSettings
      const wallet = allForms['wallet-extension'] as { transfer_enabled?: boolean } | undefined
      if (settings.leaderboard_enabled && !settings.leaderboard_balance_enabled && !settings.leaderboard_consumption_enabled && !settings.leaderboard_checkin_enabled && !settings.leaderboard_transfer_enabled) {
        return '开启排行榜时至少启用一个榜单标签'
      }
      if (settings.leaderboard_transfer_enabled && !wallet?.transfer_enabled) {
        return '转账排行榜依赖余额转账功能'
      }
      return ''
    },
  },
  {
    id: 'activity',
    placement: 'features',
    order: 20,
    component: ActivitySettingsPanel,
    settingKeys: Object.keys(defaultActivitySettings()).filter((key) => key !== 'code_format_settings_valid'),
    createForm: () => defaultActivitySettings(),
    fromSettings: (settings) => {
      const values = readSettings(defaultActivitySettings(), settings)
      const source = settings as Record<string, unknown>
      values.code_format_settings = source.code_format_settings as CodeFormatSettings ?? values.code_format_settings
      values.code_format_settings_valid = true
      return values
    },
    toPayload: (form) => {
      const settings = form as unknown as ActivitySettings
      return {
        checkin_enabled: settings.checkin_enabled,
        checkin_min_balance: Number(settings.checkin_min_balance) || 0,
        checkin_max_balance: Number(settings.checkin_max_balance) || 0,
        checkin_luck_enabled: settings.checkin_luck_enabled,
        checkin_luck_min_multiplier: Number(settings.checkin_luck_min_multiplier) || 0,
        checkin_luck_max_multiplier: Number(settings.checkin_luck_max_multiplier) || 0,
        checkin_blindbox_enabled: settings.checkin_blindbox_enabled,
        checkin_blindbox_trigger_type: settings.checkin_blindbox_trigger_type,
        checkin_blindbox_interval: Number(settings.checkin_blindbox_interval) || 1,
        code_format_settings: settings.code_format_settings,
      }
    },
    validate: (form) => {
      const settings = form as unknown as ActivitySettings
      if (!settings.code_format_settings_valid) return '兑换码格式无效'
      if (settings.checkin_enabled && Number(settings.checkin_min_balance) > Number(settings.checkin_max_balance)) return '签到最小奖励不能大于最大奖励'
      if (settings.checkin_luck_enabled && Number(settings.checkin_luck_min_multiplier) > Number(settings.checkin_luck_max_multiplier)) return '运气签到最小倍率不能大于最大倍率'
      if (settings.checkin_blindbox_enabled && !settings.checkin_enabled && !settings.checkin_luck_enabled) return '开启盲盒前至少启用一种签到方式'
      return ''
    },
  },
]
