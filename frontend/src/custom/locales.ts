import { activityAdminLocaleMessages } from './modules/activity/admin/locales'
import { activityLocaleMessages } from './modules/activity/locales'
import { gameHallAdminLocaleMessages } from './modules/game-hall/admin/locales'
import { gameHallLocaleMessages } from './modules/game-hall/locales'
import { openAIRTAdminLocaleMessages } from './modules/openai-rt/locales'
import { rechargePromotionAdminLocaleMessages, rechargePromotionLocaleMessages } from './modules/recharge-promotions/locales'
import { bepusdtAdminLocaleMessages, bepusdtLocaleMessages } from './modules/bepusdt/locales'
import { supportTicketAdminLocaleMessages, supportTicketLocaleMessages } from './modules/support-tickets/locales'
import { supportContactsLocaleMessages } from './modules/support-contacts/locales'
import { docsLocaleMessages } from './modules/docs/locales'

type LocaleTree = Record<string, unknown>

// Shared custom copy is mounted with the Overlay. It keeps custom modules from
// adding generic labels or brand wording to upstream locale source files.
const customSharedLocaleMessages = {
  en: {
    common: {
      loadMore: 'Load more',
      retry: 'Retry',
    },
  },
  zh: {
    common: {
      loadMore: '加载更多',
      retry: '重试',
    },
    nav: {
      dashboard: '系统概览',
      announcements: '通知公告',
      redeem: '兑换中心',
      proxies: '代理管理',
      redeemCodes: '兑换管理',
      promoCodes: '优惠管理',
      buySubscription: '充值订阅',
      docs: '帮助文档',
    },
  },
} as const

function isLocaleTree(value: unknown): value is LocaleTree {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
}

function mergeLocaleTrees(base: LocaleTree, extension: LocaleTree): LocaleTree {
  const merged: LocaleTree = { ...base }

  for (const [key, extensionValue] of Object.entries(extension)) {
    const baseValue = merged[key]
    merged[key] = isLocaleTree(baseValue) && isLocaleTree(extensionValue)
      ? mergeLocaleTrees(baseValue, extensionValue)
      : extensionValue
  }

  return merged
}

export const customAdminLocaleMessages = {
  en: [activityAdminLocaleMessages.en, bepusdtAdminLocaleMessages.en, gameHallAdminLocaleMessages.en, openAIRTAdminLocaleMessages.en, rechargePromotionAdminLocaleMessages.en, supportTicketAdminLocaleMessages.en],
  zh: [activityAdminLocaleMessages.zh, bepusdtAdminLocaleMessages.zh, gameHallAdminLocaleMessages.zh, openAIRTAdminLocaleMessages.zh, rechargePromotionAdminLocaleMessages.zh, supportTicketAdminLocaleMessages.zh],
} as const

export const customLocaleMessages = {
  en: [customSharedLocaleMessages.en, docsLocaleMessages.en, activityLocaleMessages.en, bepusdtLocaleMessages.en, gameHallLocaleMessages.en, rechargePromotionLocaleMessages.en, supportTicketLocaleMessages.en, supportContactsLocaleMessages.en],
  zh: [customSharedLocaleMessages.zh, docsLocaleMessages.zh, activityLocaleMessages.zh, bepusdtLocaleMessages.zh, gameHallLocaleMessages.zh, rechargePromotionLocaleMessages.zh, supportTicketLocaleMessages.zh, supportContactsLocaleMessages.zh],
} as const

export function mergeCustomLocale<T extends LocaleTree>(
  base: T,
  locale: keyof typeof customLocaleMessages,
): T {
  return customLocaleMessages[locale].reduce<LocaleTree>(mergeLocaleTrees, base) as T
}

/**
 * Applies custom admin locale fragments without replacing unrelated upstream
 * namespaces such as `admin.settings.tabs`.
 */
export function mergeCustomAdminLocale<T extends LocaleTree>(
  base: T,
  locale: keyof typeof customAdminLocaleMessages,
): T {
  return customAdminLocaleMessages[locale].reduce<LocaleTree>(mergeLocaleTrees, base) as T
}
