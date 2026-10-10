import type { Component } from 'vue'
import type { RouteRecordRaw } from 'vue-router'
import { registerFeatureFlags, type FeatureFlagDefinition } from '@/utils/featureFlags'
import { activityHeaderActions } from './modules/activity/headerActions'
import { activityHistoryTypeKey, isActivityBalanceType } from './modules/activity/ledger'
import { registrationHistoryTypeKey, isRegistrationBalanceType } from './modules/registration-protection/ledger'
import { activityNavigation } from './modules/activity/navigation'
import { activityRoutes } from './modules/activity/routes'
import {
  activityFeatureFlags,
  activitySettingsPanels,
} from './modules/activity/settings'
import { brandHomeRoutes } from './modules/brand-home/routes'
import { brandHomeSettingsPanels } from './modules/brand-home/settings'
import { gameHallNavigation } from './modules/game-hall/navigation'
import { gameHallRoutes } from './modules/game-hall/routes'
import { docsNavigation } from './modules/docs/navigation'
import { docsRoutes } from './modules/docs/routes'
import { supportTicketNavigation } from './modules/support-tickets/navigation'
import { supportTicketFeatureFlag } from './modules/support-tickets/navigation'
import { supportTicketRoutes } from './modules/support-tickets/routes'
import { supportTicketNotificationCenterExtensions } from './modules/support-tickets/notificationCenter'
import {
  gameHallFeatureFlags,
  gameHallSettingsPanels,
} from './modules/game-hall/settings'
import { openAIAccountActions, type CustomAccountAction } from './modules/openai-rt/accountActions'
import SupportContactEntry from './modules/support-contacts/SupportContactEntry.vue'
import { supportContactsSettingsPanels } from './modules/support-contacts/settings'
import { registrationProtectionSettingsPanels } from './modules/registration-protection/settings'
import { formatAuditAction } from './modules/audit-log-localization/actionLabels'
import { appleThemeSettingsPanels } from './modules/apple-theme/settings'
import { paymentDisplaySettingsPanels } from './modules/payment-display/settings'
import CheckinUserAccessModal from './modules/checkin-user-access/CheckinUserAccessModal.vue'
import { checkinUserAccessActions } from './modules/checkin-user-access/userActions'
import {
  channelMonitorV2SettingsExtensions,
  channelMonitorV2SourceExtensions,
} from './modules/channel-monitor-v2-active-probes/extensions'
import './payments'
import { xznoauthInvoiceRoutes } from './modules/xznoauth-invoice/routes'
import { xznoauthInvoiceNavigation } from './modules/xznoauth-invoice/navigation'
export {
  customPaymentDefinition,
  customPaymentDefinitions,
  customPaymentTypeDefinition,
  customPaymentTypeOptions,
  customProviderKeyOptions,
} from './paymentExtensions'
export { customPublicSettingsDefaults } from './publicSettings'

export interface CustomNavigationItem {
  path: string
  label: string
  labelKey?: string
  icon: unknown
  iconSvg?: string
  hideInSimpleMode?: boolean
  section?: 'self' | 'admin'
  slot?: 'after-batch-image' | 'after-affiliate' | 'after-transfer'
  /**
   * `false` hides the item. `undefined` keeps it visible while settings load.
   */
  featureFlag?: () => boolean | undefined
  onActivate?: () => unknown | Promise<unknown>
}

export interface CustomHeaderAction {
  id: string
  component: Component
}

export interface CustomNotificationCenterExtension {
  id: string
  component: Component
}

export interface CustomChannelMonitorV2Extension {
  id: string
  component: Component
}

export interface CustomAccountActionDefinition extends CustomAccountAction {}

export interface CustomAdminUserAction {
  id: string
  labelKey: string
  icon: 'shield'
  visible: (user: import('@/types').AdminUser) => boolean
}

export interface CustomAdminUserActionModal {
  id: string
  component: Component
}

export interface CustomRechargeSummaryExtension {
  id: string
  component: Component
}

export interface CustomPaymentOrderDetailExtension {
  id: string
  component: Component
}

export interface CustomSupportContactEntry {
  id: string
  component: Component
}

export const customSupportContactEntries: readonly CustomSupportContactEntry[] = [
  { id: 'support-contacts', component: SupportContactEntry },
]

export const customAccountActions: readonly CustomAccountActionDefinition[] = [
  ...openAIAccountActions,
]

// Core user management only knows this generic action/modal contract. Each
// downstream capability remains implemented and owned by its Overlay module.
export const customAdminUserActions: readonly CustomAdminUserAction[] = [
  ...checkinUserAccessActions,
]

export const customAdminUserActionModals: readonly CustomAdminUserActionModal[] = [
  { id: 'checkin-user-access', component: CheckinUserAccessModal },
]

// Core checkout and order pages render these components through the fixed
// generic mount points. They do not import feature implementations directly.
export const customRechargeSummaryExtensions: readonly CustomRechargeSummaryExtension[] = [
]

export const customPaymentOrderDetailExtensions: readonly CustomPaymentOrderDetailExtension[] = [
]

export const customLedgerDisplay = {
  isBalanceType: (type: string) => isActivityBalanceType(type) || isRegistrationBalanceType(type),
  historyTypeKey: (type: string) => activityHistoryTypeKey(type) ?? registrationHistoryTypeKey(type),
} as const

// Audit identifiers remain stable backend data. This Overlay only supplies the
// operator-facing label rendered by the fixed upstream audit-log mount point.
export const customAuditLogDisplay = { formatAction: formatAuditAction } as const

// This must stay structural rather than a string index signature. Intersecting
// the latter with the upstream settings form turns every upstream field into
// unknown in Vue's template type checker.
export type CustomSettingsValues = object

export interface CustomSettingsPanel {
  id: string
  placement: 'site' | 'entry-switches' | 'features'
  order: number
  component: Component
  settingKeys: readonly string[]
  createForm: () => CustomSettingsValues
  fromSettings: (settings: CustomSettingsValues) => CustomSettingsValues
  toPayload: (form: CustomSettingsValues) => CustomSettingsValues
  validate: (form: CustomSettingsValues, allForms: Readonly<Record<string, CustomSettingsValues>>) => string
}

export const customRoutes: readonly RouteRecordRaw[] = [
  ...brandHomeRoutes,
  ...docsRoutes,
  ...activityRoutes,
  ...gameHallRoutes,
  ...supportTicketRoutes,
  ...xznoauthInvoiceRoutes,
]

export const customNavigation: readonly CustomNavigationItem[] = [
  ...activityNavigation,
  ...docsNavigation,
  ...gameHallNavigation,
  ...supportTicketNavigation,
  ...xznoauthInvoiceNavigation,
]

export const customHeaderActions: readonly CustomHeaderAction[] = [
  ...activityHeaderActions,
]

export const customNotificationCenterExtensions: readonly CustomNotificationCenterExtension[] = [
  ...supportTicketNotificationCenterExtensions,
]

export const customChannelMonitorV2SettingsExtensions: readonly CustomChannelMonitorV2Extension[] = [
  ...channelMonitorV2SettingsExtensions,
]

export const customChannelMonitorV2SourceExtensions: readonly CustomChannelMonitorV2Extension[] = [
  ...channelMonitorV2SourceExtensions,
]

export const customSettingsPanels: readonly CustomSettingsPanel[] = [
  ...brandHomeSettingsPanels,
  ...appleThemeSettingsPanels,
  ...paymentDisplaySettingsPanels,
  ...gameHallSettingsPanels,
  ...activitySettingsPanels,
  ...supportContactsSettingsPanels,
  ...registrationProtectionSettingsPanels,
].sort((left, right) => left.order - right.order)

export const customFeatureFlags: readonly FeatureFlagDefinition[] = [
  ...Object.values(activityFeatureFlags),
  ...Object.values(gameHallFeatureFlags),
  supportTicketFeatureFlag,
]

registerFeatureFlags(customFeatureFlags)
