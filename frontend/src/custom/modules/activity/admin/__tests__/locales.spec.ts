import { describe, expect, it } from 'vitest'

import { activityAdminLocaleMessages } from '../locales'

describe('activity admin locale messages', () => {
  it('defines red-packet settings labels in English and Chinese', () => {
    expect(activityAdminLocaleMessages.en.settings.balanceFeatures).toMatchObject({
      redpacketEnabled: 'Enable red packets',
      redpacketMaxCount: 'Maximum red packets',
      redpacketExpireHours: 'Expiration time (hours)',
    })
    expect(activityAdminLocaleMessages.zh.settings.balanceFeatures).toMatchObject({
      redpacketEnabled: '启用红包功能',
      redpacketMaxCount: '红包最大份数',
      redpacketExpireHours: '红包过期时间（小时）',
    })
  })
})
