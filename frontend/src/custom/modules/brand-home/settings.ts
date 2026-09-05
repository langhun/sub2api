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
  const defaultHomepage = values.default_homepage === 'dino' || values.default_homepage === 'pixel'
    ? values.default_homepage
    : 'default'
  return {
    default_homepage: defaultHomepage,
  }
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
    toPayload: (form) => readSettings(form),
    validate: () => '',
  },
]
