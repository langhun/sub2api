import { describe, expect, it } from 'vitest'
import { createI18n } from 'vue-i18n'

import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'
import { customAdminLocaleMessages, customLocaleMessages } from '../locales'

describe.each([
  ['en', en, 'Dashboard', 'Check-in', 'Balance Transfer', 'General', 'Balance Features'],
  ['zh', zh, '系统概览', '签到中心', '余额转账', '通用设置', '余额功能'],
] as const)('custom %s locale integration', (
  locale,
  messages,
  dashboard,
  checkin,
  transfer,
  general,
  balanceFeatures,
) => {
  it('deeply assembles root fragments without replacing upstream navigation', () => {
    const fragments = customLocaleMessages[locale]

    for (const fragment of fragments) {
      expect(messages).toMatchObject(fragment)
    }
    expect(messages.nav).toMatchObject({ dashboard, checkin, transfer })
    expect(messages.redeem.activityHistoryTypes).toMatchObject({ checkin: locale === 'zh' ? '签到' : 'Check-in', checkin_luck: locale === 'zh' ? '运气签到' : 'Lucky check-in' })
    expect(messages.redeem.activityBalanceTypes.checkin_luck).toBe('balance')

    const i18n = createI18n({ legacy: false, locale, messages: { [locale]: messages } })
    expect(i18n.global.te('redeem.activityBalanceTypes.checkin_luck')).toBe(true)
  })

  it('deeply assembles admin fragments without replacing upstream settings', () => {
    const fragments = customAdminLocaleMessages[locale]

    for (const fragment of fragments) {
      expect(messages.admin).toMatchObject(fragment)
    }
    expect(messages.admin.settings.tabs).toMatchObject({ general, balanceFeatures })
  })
})
