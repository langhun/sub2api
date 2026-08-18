import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import RegistrationProtectionSettingsPanel from './RegistrationProtectionSettingsPanel.vue'

export interface RegistrationProtectionSettings {
  registration_alias_protection_enabled: boolean
  registration_reward_fingerprint_protection_enabled: boolean
}

export const defaultRegistrationProtectionSettings = (): RegistrationProtectionSettings => ({
  registration_alias_protection_enabled: true,
  registration_reward_fingerprint_protection_enabled: true,
})

function booleanSetting(value: unknown, fallback: boolean): boolean {
  return typeof value === 'boolean' ? value : fallback
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
  }
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
        registration_alias_protection_enabled: settings.registration_alias_protection_enabled,
        registration_reward_fingerprint_protection_enabled: settings.registration_reward_fingerprint_protection_enabled,
      }
    },
    validate: () => '',
  },
]
