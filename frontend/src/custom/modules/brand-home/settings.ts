import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import BrandHomeSettingsPanel from './admin/BrandHomeSettingsPanel.vue'

interface BrandHomeSettings {
  default_homepage: 'default' | 'dino' | 'pixel'
}

const defaultBrandHomeSettings = (): BrandHomeSettings => ({
  default_homepage: 'default',
})

function readSettings(settings: CustomSettingsValues): BrandHomeSettings {
  const values = settings as Record<string, unknown>
  if (values.default_homepage === 'dino' || values.default_homepage === 'pixel') {
    return { default_homepage: values.default_homepage }
  }
  return defaultBrandHomeSettings()
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
