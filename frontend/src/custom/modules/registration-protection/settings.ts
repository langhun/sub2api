import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import RegistrationProtectionSettingsPanel from './RegistrationProtectionSettingsPanel.vue'

export interface RegistrationProtectionSettings {
  registration_alias_protection_enabled: boolean
  registration_reward_fingerprint_protection_enabled: boolean
  registration_cluster_protection_enabled: boolean
  registration_cluster_window_6h: number
  registration_cluster_window_24h: number
  registration_cluster_medium_threshold: number
  registration_cluster_high_threshold: number
  registration_auto_high_risk_lock_enabled: boolean
  registration_trial_enabled: boolean
  registration_trial_hours: number
  registration_trial_cost_limit: number
  registration_trial_rpm_limit: number
  registration_trial_concurrency: number
  registration_payment_unlock_enabled: boolean
  registration_payment_unlock_amount: number
}

export const defaultRegistrationProtectionSettings = (): RegistrationProtectionSettings => ({
  registration_alias_protection_enabled: true,
  registration_reward_fingerprint_protection_enabled: true,
  registration_cluster_protection_enabled: true,
  registration_cluster_window_6h: 3,
  registration_cluster_window_24h: 5,
  registration_cluster_medium_threshold: 3,
  registration_cluster_high_threshold: 5,
  registration_auto_high_risk_lock_enabled: true,
  registration_trial_enabled: true,
  registration_trial_hours: 24,
  registration_trial_cost_limit: 2,
  registration_trial_rpm_limit: 10,
  registration_trial_concurrency: 2,
  registration_payment_unlock_enabled: true,
  registration_payment_unlock_amount: 10,
})

function booleanSetting(value: unknown, fallback: boolean): boolean {
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (normalized === 'true') return true
    if (normalized === 'false') return false
  }
  return fallback
}

export function readRegistrationProtectionSettings(settings: CustomSettingsValues): RegistrationProtectionSettings {
  const values = settings as Record<string, unknown>
  const defaults = defaultRegistrationProtectionSettings()
  return {
    registration_alias_protection_enabled: booleanSetting(
      values.registration_alias_protection_enabled,
      defaults.registration_alias_protection_enabled,
    ),
    registration_reward_fingerprint_protection_enabled: booleanSetting(
      values.registration_reward_fingerprint_protection_enabled,
      defaults.registration_reward_fingerprint_protection_enabled,
    ),
    registration_cluster_protection_enabled: booleanSetting(values.registration_cluster_protection_enabled, defaults.registration_cluster_protection_enabled),
    registration_cluster_window_6h: numberSetting(values.registration_cluster_window_6h, defaults.registration_cluster_window_6h),
    registration_cluster_window_24h: numberSetting(values.registration_cluster_window_24h, defaults.registration_cluster_window_24h),
    registration_cluster_medium_threshold: numberSetting(values.registration_cluster_medium_threshold, defaults.registration_cluster_medium_threshold),
    registration_cluster_high_threshold: numberSetting(values.registration_cluster_high_threshold, defaults.registration_cluster_high_threshold),
    registration_auto_high_risk_lock_enabled: booleanSetting(values.registration_auto_high_risk_lock_enabled, defaults.registration_auto_high_risk_lock_enabled),
    registration_trial_enabled: booleanSetting(values.registration_trial_enabled, defaults.registration_trial_enabled),
    registration_trial_hours: numberSetting(values.registration_trial_hours, defaults.registration_trial_hours),
    registration_trial_cost_limit: numberSetting(values.registration_trial_cost_limit, defaults.registration_trial_cost_limit),
    registration_trial_rpm_limit: numberSetting(values.registration_trial_rpm_limit, defaults.registration_trial_rpm_limit),
    registration_trial_concurrency: numberSetting(values.registration_trial_concurrency, defaults.registration_trial_concurrency),
    registration_payment_unlock_enabled: booleanSetting(values.registration_payment_unlock_enabled, defaults.registration_payment_unlock_enabled),
    registration_payment_unlock_amount: numberSetting(values.registration_payment_unlock_amount, defaults.registration_payment_unlock_amount),
  }
}

function numberSetting(value: unknown, fallback: number): number {
  return typeof value === 'number' && Number.isFinite(value) && value > 0 ? value : fallback
}

export const registrationProtectionSettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'registration-protection',
    placement: 'features',
    order: 35,
    component: RegistrationProtectionSettingsPanel,
    settingKeys: Object.keys(defaultRegistrationProtectionSettings()),
    createForm: defaultRegistrationProtectionSettings,
    fromSettings: readRegistrationProtectionSettings,
    toPayload: (form) => {
      const settings = readRegistrationProtectionSettings(form)
      return {
        ...settings,
      }
    },
    validate: (form) => validateRegistrationProtectionSettings(readRegistrationProtectionSettings(form)),
  },
]

export function validateRegistrationProtectionSettings(settings: RegistrationProtectionSettings): string {
  const positive = [
    settings.registration_cluster_window_6h,
    settings.registration_cluster_window_24h,
    settings.registration_cluster_medium_threshold,
    settings.registration_cluster_high_threshold,
    settings.registration_trial_hours,
    settings.registration_trial_cost_limit,
    settings.registration_trial_rpm_limit,
    settings.registration_trial_concurrency,
    settings.registration_payment_unlock_amount,
  ]
  if (positive.some((value) => !Number.isFinite(value) || value <= 0)) return '注册风控限制必须是大于 0 的有效数字'
  if (settings.registration_cluster_window_24h < settings.registration_cluster_window_6h) return '24 小时网段窗口不能小于 6 小时窗口'
  if (settings.registration_cluster_high_threshold < settings.registration_cluster_medium_threshold) return '高风险阈值不能小于中风险阈值'
  if (settings.registration_cluster_window_6h > 1000 || settings.registration_cluster_window_24h > 5000 || settings.registration_cluster_medium_threshold > 100 || settings.registration_cluster_high_threshold > 100) return '网段聚集限制超过安全上限'
  if (settings.registration_trial_hours > 168 || settings.registration_trial_rpm_limit > 600 || settings.registration_trial_concurrency > 32 || settings.registration_trial_cost_limit > 10000 || settings.registration_payment_unlock_amount > 100000) return '试用限制超过安全上限'
  return ''
}
