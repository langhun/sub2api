import { describe, expect, it } from 'vitest'

import enAdminAccounts from '../locales/en/admin/accounts'
import enAdminChannels from '../locales/en/admin/channels'
import enAdminOps from '../locales/en/admin/ops'
import enAdminOverview from '../locales/en/admin/overview'
import enAdminResources from '../locales/en/admin/resources'
import enAdminSettings from '../locales/en/admin/settings'
import zhAdminAccounts from '../locales/zh/admin/accounts'
import zhAdminChannels from '../locales/zh/admin/channels'
import zhAdminOps from '../locales/zh/admin/ops'
import zhAdminOverview from '../locales/zh/admin/overview'
import zhAdminResources from '../locales/zh/admin/resources'
import zhAdminSettings from '../locales/zh/admin/settings'
import en from '../locales/en'
import zh from '../locales/zh'
import { customAdminLocaleMessages, customLocaleMessages } from '@/custom/locales'

// 根语言包和 admin 语言包使用受控深合并；同名顶层键必须经过合并测试，不能依赖对象展开的覆盖顺序。
type Modules = Record<string, Record<string, unknown>>

function collisions(modules: Modules): string[] {
  const seen = new Map<string, string>()
  const out: string[] = []
  for (const [name, mod] of Object.entries(modules)) {
    for (const key of Object.keys(mod)) {
      const prev = seen.get(key)
      if (prev) {
        out.push(`"${key}" in both ${prev} and ${name}`)
      } else {
        seen.set(key, name)
      }
    }
  }
  return out
}

const admins: Record<string, Modules> = {
  zh: {
    overview: zhAdminOverview,
    channels: zhAdminChannels,
    accounts: zhAdminAccounts,
    resources: zhAdminResources,
    ops: zhAdminOps,
    settings: zhAdminSettings
  },
  en: {
    overview: enAdminOverview,
    channels: enAdminChannels,
    accounts: enAdminAccounts,
    resources: enAdminResources,
    ops: enAdminOps,
    settings: enAdminSettings
  }
}

describe.each(['zh', 'en'] as const)('locale %s overlay assembly', (locale) => {
  it('deeply assembles custom root fragments without replacing upstream navigation', () => {
    const messages = locale === 'zh' ? zh : en
    const fragments = customLocaleMessages[locale as keyof typeof customLocaleMessages]

    expect(messages).toMatchObject(fragments[0])
    expect(messages).toMatchObject(fragments[1])
    expect(messages).toMatchObject(fragments[2])
    expect(messages.nav).toMatchObject({
      dashboard: locale === 'zh' ? '系统概览' : 'Dashboard',
      checkin: locale === 'zh' ? '签到中心' : 'Check-in',
      transfer: locale === 'zh' ? '余额转账' : 'Balance Transfer',
    })
  })

  it('deeply assembles custom admin fragments without replacing upstream settings', () => {
    const messages = locale === 'zh' ? zh : en
    const fragments = customAdminLocaleMessages[locale as keyof typeof customAdminLocaleMessages]

    expect(messages.admin).toMatchObject(fragments[0])
    expect(messages.admin).toMatchObject(fragments[1])
    expect(messages.admin).toMatchObject(fragments[2])
    expect(messages.admin.settings.tabs).toMatchObject({
      general: locale === 'zh' ? '通用设置' : 'General',
      balanceFeatures: locale === 'zh' ? '余额功能' : 'Balance Features',
    })
  })

  it('admin modules have no overlapping top-level keys', () => {
    expect(collisions(admins[locale])).toEqual([])
  })
})
