import { activityAdminLocaleMessages } from './modules/activity/admin/locales'
import { activityLocaleMessages } from './modules/activity/locales'
import { gameHallAdminLocaleMessages } from './modules/game-hall/admin/locales'
import { gameHallLocaleMessages } from './modules/game-hall/locales'
import { walletExtensionAdminLocaleMessages } from './modules/wallet-extension/admin/locales'
import { walletExtensionLocaleMessages } from './modules/wallet-extension/locales'

type LocaleTree = Record<string, unknown>

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
  en: [activityAdminLocaleMessages.en, gameHallAdminLocaleMessages.en, walletExtensionAdminLocaleMessages.en],
  zh: [activityAdminLocaleMessages.zh, gameHallAdminLocaleMessages.zh, walletExtensionAdminLocaleMessages.zh],
} as const

export const customLocaleMessages = {
  en: [activityLocaleMessages.en, gameHallLocaleMessages.en, walletExtensionLocaleMessages.en],
  zh: [activityLocaleMessages.zh, gameHallLocaleMessages.zh, walletExtensionLocaleMessages.zh],
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
