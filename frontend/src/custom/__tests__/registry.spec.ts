import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import {
  customHeaderActions,
  customAccountMenuActions,
  customNavigation,
  customRoutes,
  customSettingsPanels,
} from '../registry'
import { activityHeaderActions } from '../modules/activity/headerActions'
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
const headerSource = readFileSync(resolve(directory, '../../components/layout/AppHeader.vue'), 'utf8')
const settingsSource = readFileSync(resolve(directory, '../../views/admin/SettingsView.vue'), 'utf8')
const accountMenuSource = readFileSync(resolve(directory, '../../components/admin/account/AccountActionMenu.vue'), 'utf8')

describe('custom overlay registry', () => {
  it('aggregates routes and navigation from custom modules', () => {
    expect(customRoutes).toEqual(expect.arrayContaining(brandHomeRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(activityRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(gameHallRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(walletExtensionRoutes))
    expect(customNavigation).toEqual(expect.arrayContaining(activityNavigation))
    expect(customNavigation).toEqual(expect.arrayContaining(gameHallNavigation))
    expect(customNavigation).toEqual(expect.arrayContaining(walletExtensionNavigation))
    expect(customHeaderActions).toEqual(expect.arrayContaining(activityHeaderActions))
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

  it('leaves usage routes and the sidebar entry to the activity module', () => {
    expect(routerSource).not.toContain("path: '/usage'")
    expect(routerSource).not.toContain("path: '/key-usage'")
    expect(sidebarSource).toContain("...buildCustomNavigationItems('after-batch-image')")
    expect(sidebarSource).not.toContain("{ path: '/usage'")
    expect(customRoutes.filter((route) => route.path === '/usage')).toHaveLength(1)
    expect(customRoutes.filter((route) => route.path === '/key-usage')).toHaveLength(1)
    expect(customNavigation).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: '/usage', slot: 'after-batch-image' }),
    ]))
  })

  it('mounts custom navigation through the sidebar self-navigation builder', () => {
    expect(sidebarSource).toContain('function buildCustomNavigationItems(')
    expect(sidebarSource).toContain("...buildCustomNavigationItems('after-affiliate')")
    expect(sidebarSource).toContain('label: item.labelKey ? t(item.labelKey) : item.label')
  })

  it('mounts header actions through the registry without importing activity UI directly', () => {
    expect(headerSource).toContain("import { customHeaderActions } from '@/custom/registry'")
    expect(headerSource).toContain('v-for="action in customHeaderActions"')
    expect(headerSource).not.toContain("@/custom/modules/activity/components/CheckinHeaderActions.vue")
  })

  it('mounts custom account actions through the account menu', () => {
    expect(customAccountMenuActions.map((action) => action.id)).toContain('account-drain-target')
    expect(accountMenuSource).toContain("import { customAccountMenuActions } from '@/custom/registry'")
    expect(accountMenuSource).toContain('v-for="action in customAccountMenuActions"')
    expect(accountMenuSource).not.toContain("@/custom/modules/account-drain/components/AccountDrainMenuAction.vue")
  })

  it('owns custom setting panels through the registry rather than the shared form', () => {
    expect(customSettingsPanels.map((panel) => panel.id)).toEqual([
      'brand-home',
      'game-hall',
      'activity-entry-switches',
      'activity',
      'wallet-extension',
    ])
    expect(settingsSource).toContain("customSettingsPanelsFor('site')")
    expect(settingsSource).toContain("customSettingsPanelsFor('entry-switches')")
    expect(settingsSource).toContain("customSettingsPanelsFor('features')")
    expect(settingsSource).toContain('panel.toPayload(customSettingsForms[panel.id] ?? {})')
    expect(settingsSource).not.toContain('form.game_hall_enabled')
    expect(settingsSource).not.toContain('form.checkin_enabled')
    expect(settingsSource).not.toContain('form.transfer_enabled')
    expect(settingsSource).not.toContain('form.default_homepage')
  })
})
