import { activityAdminLocaleMessages } from './modules/activity/admin/locales'
import { walletExtensionAdminLocaleMessages } from './modules/wallet-extension/admin/locales'

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
  en: [activityAdminLocaleMessages.en, walletExtensionAdminLocaleMessages.en],
  zh: [activityAdminLocaleMessages.zh, walletExtensionAdminLocaleMessages.zh],
} as const

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
