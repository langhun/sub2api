import { describe, expect, it } from 'vitest'

import { brandHomeSettingsPanels } from '../settings'

describe('brand-home settings', () => {
  const panel = brandHomeSettingsPanels[0]!

  it('defaults the site theme to liquid glass', () => {
    expect(panel.createForm()).toEqual({
      default_homepage: 'default',
      theme_style: 'glass',
    })
  })

  it('round-trips supported theme values in the admin payload', () => {
    expect(panel.settingKeys).toContain('theme_style')
    expect(panel.toPayload({ default_homepage: 'pixel', theme_style: 'classic' })).toEqual({
      default_homepage: 'pixel',
      theme_style: 'classic',
    })
  })

  it('falls back to liquid glass for unsupported values', () => {
    expect(panel.fromSettings({ default_homepage: 'invalid', theme_style: 'invalid' })).toEqual({
      default_homepage: 'default',
      theme_style: 'glass',
    })
  })
})
