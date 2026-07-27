import { describe, expect, it } from 'vitest'
import { mergeCustomAdminLocale, mergeCustomLocale } from '../locales'

describe('custom locale overlay', () => {
  it('deeply merges navigation fragments without replacing upstream siblings', () => {
    const base = {
      nav: { dashboard: 'Dashboard' },
      common: { retry: 'Retry' },
    }

    const merged = mergeCustomLocale(base, 'en')

    expect(merged.nav).toMatchObject({
      dashboard: 'Dashboard',
      checkin: 'Check-in',
      transfer: 'Balance Transfer',
    })
    expect(merged.common).toEqual({ retry: 'Retry', loadMore: 'Load more' })
    expect(base.nav).toEqual({ dashboard: 'Dashboard' })
  })
})

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
    expect(merged.audit).toMatchObject({
      title: 'Audit Logs',
      actionSegments: { checkin: 'Check-in', transfer: 'Transfer' },
    })
    expect(merged.settings.balanceFeatures).toMatchObject({
      checkinTitle: 'Check-in Settings',
      transferTitle: 'Transfers and Red Packets',
    })
    expect(base.settings.tabs).toEqual({ general: 'General' })
  })
})
