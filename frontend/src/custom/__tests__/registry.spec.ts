import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import { customNavigation, customRoutes } from '../registry'
import { activityNavigation } from '../modules/activity/navigation'
import { activityRoutes } from '../modules/activity/routes'
import { brandHomeRoutes } from '../modules/brand-home/routes'
import { gameHallNavigation } from '../modules/game-hall/navigation'
import { gameHallRoutes } from '../modules/game-hall/routes'
import { walletExtensionNavigation } from '../modules/wallet-extension/navigation'
import { walletExtensionRoutes } from '../modules/wallet-extension/routes'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../../router/index.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(directory, '../../components/layout/AppSidebar.vue'), 'utf8')

describe('custom overlay registry', () => {
  it('aggregates routes and navigation from custom modules', () => {
    expect(customRoutes).toEqual(expect.arrayContaining(brandHomeRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(activityRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(gameHallRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(walletExtensionRoutes))
    expect(customNavigation).toEqual(expect.arrayContaining(activityNavigation))
    expect(customNavigation).toEqual(expect.arrayContaining(gameHallNavigation))
    expect(customNavigation).toEqual(expect.arrayContaining(walletExtensionNavigation))
  })

  it('mounts custom routes before the catch-all route', () => {
    expect(routerSource.indexOf('...customRoutes')).toBeGreaterThan(-1)
    expect(routerSource.indexOf('...customRoutes')).toBeLessThan(routerSource.indexOf("path: '/:pathMatch(.*)*'"))
  })

  it('leaves the direct-transfer URL to the wallet extension', () => {
    expect(routerSource).not.toContain("path: '/transfer'")
    expect(sidebarSource).not.toContain("{ path: '/transfer'")
    expect(customRoutes.filter((route) => route.path === '/transfer')).toHaveLength(1)
    expect(customNavigation.filter((item) => item.path === '/transfer')).toHaveLength(1)
  })

  it('mounts custom navigation through the sidebar self-navigation builder', () => {
    expect(sidebarSource).toContain('function buildCustomNavigationItems(')
    expect(sidebarSource).toContain("...buildCustomNavigationItems('after-affiliate')")
    expect(sidebarSource).toContain('label: item.labelKey ? t(item.labelKey) : item.label')
  })
})
