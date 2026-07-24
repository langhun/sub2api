import type { RouteRecordRaw } from 'vue-router'

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

export const customRoutes: readonly RouteRecordRaw[] = []

export const customNavigation: readonly CustomNavigationItem[] = []
