import type { RouteRecordRaw } from 'vue-router'
import { activityNavigation } from './modules/activity/navigation'
import { activityRoutes } from './modules/activity/routes'
import { brandHomeRoutes } from './modules/brand-home/routes'
import { gameHallNavigation } from './modules/game-hall/navigation'
import { gameHallRoutes } from './modules/game-hall/routes'
import { walletExtensionNavigation } from './modules/wallet-extension/navigation'
import { walletExtensionRoutes } from './modules/wallet-extension/routes'

export interface CustomNavigationItem {
  path: string
  label: string
  labelKey?: string
  icon: unknown
  iconSvg?: string
  hideInSimpleMode?: boolean
  slot?: 'after-affiliate' | 'after-transfer'
  /**
   * `false` hides the item. `undefined` keeps it visible while settings load.
   */
  featureFlag?: () => boolean | undefined
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
