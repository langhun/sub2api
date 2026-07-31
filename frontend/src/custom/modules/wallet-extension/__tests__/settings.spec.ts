import { describe, expect, it } from 'vitest'
import { walletExtensionSettingsPanels } from '../settings'

describe('wallet extension settings panel', () => {
  it('keeps values returned by the admin settings API', () => {
    const form = walletExtensionSettingsPanels[0].fromSettings({
      transfer_enabled: true,
      transfer_daily_limit: 321,
    }) as Record<string, unknown>

    expect(form.transfer_enabled).toBe(true)
    expect(form.transfer_daily_limit).toBe(321)
  })

  it('does not own red-packet settings', () => {
    const panel = walletExtensionSettingsPanels[0]
    const form = panel.fromSettings({
      transfer_enabled: true,
      redpacket_enabled: true,
      redpacket_max_count: 9,
    }) as Record<string, unknown>

    expect(panel.settingKeys).not.toContain('redpacket_enabled')
    expect(form).not.toHaveProperty('redpacket_enabled')
    expect(panel.toPayload(form)).not.toHaveProperty('redpacket_enabled')
  })
})
