import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import BrandHomeSettingsPanel from './admin/BrandHomeSettingsPanel.vue'

interface BrandHomeSettings {
  default_homepage: 'default' | 'dino'
}

const defaultBrandHomeSettings = (): BrandHomeSettings => ({
  default_homepage: 'default',
})

function readSettings(settings: CustomSettingsValues): BrandHomeSettings {
  return settings.default_homepage === 'dino'
    ? { default_homepage: 'dino' }
    : defaultBrandHomeSettings()
}

export const brandHomeSettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'brand-home',
    placement: 'site',
    order: 10,
    component: BrandHomeSettingsPanel,
    settingKeys: ['default_homepage'],
    createForm: defaultBrandHomeSettings,
    fromSettings: readSettings,
    toPayload: (form) => ({ default_homepage: readSettings(form).default_homepage }),
    validate: () => '',
  },
]
