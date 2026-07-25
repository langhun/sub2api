import type { Component } from 'vue'
import type { RouteRecordRaw } from 'vue-router'
import { registerFeatureFlags, type FeatureFlagDefinition } from '@/utils/featureFlags'
import { activityHeaderActions } from './modules/activity/headerActions'
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
import {
  gameHallFeatureFlags,
  gameHallSettingsPanels,
} from './modules/game-hall/settings'
import { walletExtensionNavigation } from './modules/wallet-extension/navigation'
import { walletExtensionRoutes } from './modules/wallet-extension/routes'
import {
  walletExtensionFeatureFlags,
  walletExtensionSettingsPanels,
} from './modules/wallet-extension/settings'
export { customPublicSettingsDefaults } from './publicSettings'

export interface CustomNavigationItem {
  path: string
  label: string
  labelKey?: string
  icon: unknown
  iconSvg?: string
  hideInSimpleMode?: boolean
  section?: 'self' | 'admin'
  slot?: 'after-affiliate' | 'after-transfer'
  /**
   * `false` hides the item. `undefined` keeps it visible while settings load.
   */
  featureFlag?: () => boolean | undefined
}

export interface CustomHeaderAction {
  id: string
  component: Component
}

export type CustomSettingsValues = Record<string, unknown>

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
  ...activityRoutes,
  ...gameHallRoutes,
  ...walletExtensionRoutes,
]

export const customNavigation: readonly CustomNavigationItem[] = [
  ...activityNavigation,
  ...gameHallNavigation,
  ...walletExtensionNavigation,
]

export const customHeaderActions: readonly CustomHeaderAction[] = [
  ...activityHeaderActions,
]

export const customSettingsPanels: readonly CustomSettingsPanel[] = [
  ...brandHomeSettingsPanels,
  ...gameHallSettingsPanels,
  ...activitySettingsPanels,
  ...walletExtensionSettingsPanels,
].sort((left, right) => left.order - right.order)

export const customFeatureFlags: readonly FeatureFlagDefinition[] = [
  ...Object.values(activityFeatureFlags),
  ...Object.values(gameHallFeatureFlags),
  ...Object.values(walletExtensionFeatureFlags),
]

registerFeatureFlags(customFeatureFlags)
