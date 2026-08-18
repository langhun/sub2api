import { createI18n } from 'vue-i18n'
import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import RegistrationProtectionSettingsPanel from '../RegistrationProtectionSettingsPanel.vue'
import {
  defaultRegistrationProtectionSettings,
  readRegistrationProtectionSettings,
  registrationProtectionSettingsPanels,
} from '../settings'

describe('registration protection settings', () => {
  it('defaults both protections to enabled', () => {
    expect(defaultRegistrationProtectionSettings()).toEqual({
      registration_alias_protection_enabled: true,
      registration_reward_fingerprint_protection_enabled: true,
    })
  })

  it('keeps explicit values and applies defaults for missing values', () => {
    expect(readRegistrationProtectionSettings({
      registration_alias_protection_enabled: false,
    })).toEqual({
      registration_alias_protection_enabled: false,
      registration_reward_fingerprint_protection_enabled: true,
    })
  })

  it('serializes both settings for the admin settings API', () => {
    const panel = registrationProtectionSettingsPanels[0]
    expect(panel.toPayload({
      registration_alias_protection_enabled: false,
      registration_reward_fingerprint_protection_enabled: true,
    })).toEqual({
      registration_alias_protection_enabled: false,
      registration_reward_fingerprint_protection_enabled: true,
    })
  })

  it('renders both switches and emits changes', async () => {
    const wrapper = mount(RegistrationProtectionSettingsPanel, {
      props: {
        modelValue: {
          registration_alias_protection_enabled: true,
          registration_reward_fingerprint_protection_enabled: false,
        },
      },
      global: {
        plugins: [createI18n({ legacy: false, locale: 'en', messages: { en: {} } })],
      },
    })

    const switches = wrapper.findAll('[role="switch"]')
    expect(switches).toHaveLength(2)
    expect(switches[0].attributes('aria-checked')).toBe('true')
    expect(switches[1].attributes('aria-checked')).toBe('false')

    await switches[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual({
      registration_alias_protection_enabled: true,
      registration_reward_fingerprint_protection_enabled: true,
    })
  })
})
