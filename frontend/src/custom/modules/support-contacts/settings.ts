import type { CustomSettingsPanel, CustomSettingsValues } from '../../registry'
import SupportContactsSettingsPanel from './SupportContactsSettingsPanel.vue'

export const supportContactsSettingsPanels: readonly CustomSettingsPanel[] = [
  {
    id: 'support-contacts',
    placement: 'site',
    order: 40,
    component: SupportContactsSettingsPanel,
    settingKeys: [],
    createForm: () => ({}),
    fromSettings: () => ({}),
    toPayload: () => ({}),
    validate: (_form: CustomSettingsValues) => '',
  },
]
