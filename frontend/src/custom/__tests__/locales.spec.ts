import { describe, expect, it } from 'vitest'
import { mergeCustomAdminLocale } from '../locales'

describe('custom admin locale overlay', () => {
  it('deeply merges custom fragments without replacing upstream siblings', () => {
    const base = {
      settings: {
        tabs: { general: 'General' },
        features: { channelMonitor: 'Channel Monitor' },
      },
      audit: { title: 'Audit Logs' },
    }

    const merged = mergeCustomAdminLocale(base, 'en')

    expect(merged.settings.tabs).toMatchObject({
      general: 'General',
      balanceFeatures: 'Balance Features',
    })
    expect(merged.settings.features).toEqual({ channelMonitor: 'Channel Monitor' })
    expect(merged.audit).toEqual({ title: 'Audit Logs' })
    expect(merged.settings.balanceFeatures).toMatchObject({
      checkinTitle: 'Check-in Settings',
      transferTitle: 'Transfers and Red Packets',
    })
    expect(base.settings.tabs).toEqual({ general: 'General' })
  })
})
