import { defineFeatureFlag } from '@/utils/featureFlags'
import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import WalletExtensionSettingsPanel from './admin/WalletExtensionSettingsPanel.vue'
export { walletExtensionPublicSettingsDefaults } from './publicSettings'

export const walletExtensionFeatureFlags = {
  transfer: defineFeatureFlag({ key: 'transfer_enabled', mode: 'opt-in', label: 'Balance Transfer' }),
} as const

interface WalletExtensionSettings {
  transfer_enabled: boolean
  transfer_fee_rate: number
  transfer_min_amount: number
  transfer_max_amount: number
  transfer_daily_limit: number
  transfer_daily_count_limit: number
  transfer_vip_fee_exempt: boolean
}

const defaultWalletExtensionSettings = (): WalletExtensionSettings => ({
  transfer_enabled: false,
  transfer_fee_rate: 0.01,
  transfer_min_amount: 0.01,
  transfer_max_amount: 1000,
  transfer_daily_limit: 1000,
  transfer_daily_count_limit: 50,
  transfer_vip_fee_exempt: false,
})

function readSettings(settings: CustomSettingsValues): WalletExtensionSettings {
  const defaults = defaultWalletExtensionSettings()
  const values = settings as Record<string, unknown>
  const result = defaults as unknown as Record<string, unknown>
  for (const key of Object.keys(defaults) as Array<keyof WalletExtensionSettings>) {
    const value = values[key]
    if (value !== null && value !== undefined) result[key] = value
  }
  return result as unknown as WalletExtensionSettings
}

export const walletExtensionSettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'wallet-extension',
    placement: 'features',
    order: 30,
    component: WalletExtensionSettingsPanel,
    settingKeys: Object.keys(defaultWalletExtensionSettings()),
    createForm: () => defaultWalletExtensionSettings(),
    fromSettings: readSettings,
    toPayload: (form) => {
      const settings = form as unknown as WalletExtensionSettings
      return {
        transfer_enabled: settings.transfer_enabled,
        transfer_fee_rate: Number(settings.transfer_fee_rate) || 0,
        transfer_min_amount: Number(settings.transfer_min_amount) || 0,
        transfer_max_amount: Number(settings.transfer_max_amount) || 0,
        transfer_daily_limit: Number(settings.transfer_daily_limit) || 0,
        transfer_daily_count_limit: Number(settings.transfer_daily_count_limit) || 0,
        transfer_vip_fee_exempt: settings.transfer_vip_fee_exempt,
      }
    },
    validate: (form) => {
      const settings = form as unknown as WalletExtensionSettings
      return settings.transfer_enabled && Number(settings.transfer_min_amount) > Number(settings.transfer_max_amount)
        ? '转账最小金额不能大于最大金额'
        : ''
    },
  },
]
