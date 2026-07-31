import { describe, expect, it } from 'vitest'

import { activityPublicSettingsDefaults } from '../publicSettings'
import { activityFeatureFlags, activitySettingsPanels } from '../settings'

describe('activity settings ownership', () => {
  it('owns red-packet settings through the activity panel', () => {
    const panel = activitySettingsPanels.find(({ id }) => id === 'activity')
    expect(panel).toBeDefined()

    const form = panel!.fromSettings({
      redpacket_enabled: true,
      redpacket_max_count: 9,
      redpacket_expire_hours: 12,
    }) as Record<string, unknown>

    expect(panel!.settingKeys).toEqual(expect.arrayContaining([
      'redpacket_enabled',
      'redpacket_max_count',
      'redpacket_expire_hours',
    ]))
    expect(form).toMatchObject({
      redpacket_enabled: true,
      redpacket_max_count: 9,
      redpacket_expire_hours: 12,
    })
    expect(panel!.toPayload(form)).toMatchObject({
      redpacket_enabled: true,
      redpacket_max_count: 9,
      redpacket_expire_hours: 12,
    })
  })

  it('publishes the red-packet opt-in switch from activity', () => {
    expect(activityFeatureFlags.redpacket).toMatchObject({
      key: 'redpacket_enabled',
      mode: 'opt-in',
    })
    expect(activityPublicSettingsDefaults.redpacket_enabled).toBe(false)
  })
})
