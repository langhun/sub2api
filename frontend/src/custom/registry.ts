import type { RouteRecordRaw } from 'vue-router'
import { brandHomeRoutes } from './modules/brand-home/routes'

export interface CustomNavigationItem {
  path: string
  label: string
  icon: unknown
  iconSvg?: string
  hideInSimpleMode?: boolean
  /**
   * `false` hides the item. `undefined` keeps it visible while settings load.
   */
  featureFlag?: () => boolean | undefined
}

export const customRoutes: readonly RouteRecordRaw[] = [
  ...brandHomeRoutes,
]

export const customNavigation: readonly CustomNavigationItem[] = []
