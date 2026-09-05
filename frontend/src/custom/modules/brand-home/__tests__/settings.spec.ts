import { describe, expect, it } from 'vitest'

import { brandHomeSettingsPanels } from '../settings'

describe('brand-home settings', () => {
  const panel = brandHomeSettingsPanels[0]!

  it('defaults the homepage independently from theme settings', () => {
    expect(panel.createForm()).toEqual({
      default_homepage: 'default',
    })
  })

  it('round-trips supported homepage values without owning theme keys', () => {
    expect(panel.settingKeys).not.toContain('theme_style')
    expect(panel.toPayload({ default_homepage: 'pixel' })).toEqual({
      default_homepage: 'pixel',
    })
  })

  it('falls back to the default homepage for unsupported values', () => {
    expect(panel.fromSettings({ default_homepage: 'invalid', theme_style: 'invalid' })).toEqual({
      default_homepage: 'default',
    })
  })
})
