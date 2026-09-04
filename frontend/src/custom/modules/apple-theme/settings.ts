import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import AppleThemeSettingsPanel from './admin/AppleThemeSettingsPanel.vue'

interface AppleThemeSettings {
  enable_apple_theme: boolean
  default_theme_mode: 'light' | 'dark' | 'auto'
}

const defaultAppleThemeSettings = (): AppleThemeSettings => ({
  enable_apple_theme: false,
  default_theme_mode: 'auto',
})

function readSettings(settings: CustomSettingsValues): AppleThemeSettings {
  const values = settings as Record<string, unknown>
  return {
    enable_apple_theme: values.enable_apple_theme === true,
    default_theme_mode:
      values.default_theme_mode === 'light' ||
      values.default_theme_mode === 'dark' ||
      values.default_theme_mode === 'auto'
        ? values.default_theme_mode
        : 'auto',
  }
}

export const appleThemeSettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'apple-theme',
    placement: 'site',
    order: 15,
    component: AppleThemeSettingsPanel,
    settingKeys: ['enable_apple_theme', 'default_theme_mode'],
    createForm: defaultAppleThemeSettings,
    fromSettings: readSettings,
    toPayload: (form) => readSettings(form),
    validate: () => '',
  },
]
