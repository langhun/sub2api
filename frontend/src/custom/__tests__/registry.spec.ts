import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import { customNavigation, customRoutes } from '../registry'
import { brandHomeRoutes } from '../modules/brand-home/routes'
import { gameHallNavigation } from '../modules/game-hall/navigation'
import { gameHallRoutes } from '../modules/game-hall/routes'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../../router/index.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(directory, '../../components/layout/AppSidebar.vue'), 'utf8')

describe('custom overlay registry', () => {
  it('aggregates routes and navigation from custom modules', () => {
    expect(customRoutes).toEqual(expect.arrayContaining(brandHomeRoutes))
    expect(customRoutes).toEqual(expect.arrayContaining(gameHallRoutes))
    expect(customNavigation).toEqual(expect.arrayContaining(gameHallNavigation))
  })

  it('mounts custom routes before the catch-all route', () => {
    expect(routerSource.indexOf('...customRoutes')).toBeGreaterThan(-1)
    expect(routerSource.indexOf('...customRoutes')).toBeLessThan(routerSource.indexOf("path: '/:pathMatch(.*)*'"))
  })

  it('mounts custom navigation through the sidebar self-navigation builder', () => {
    expect(sidebarSource).toContain('...customNavigation.map((item): NavItem => ({')
    expect(sidebarSource).toContain('label: item.labelKey ? t(item.labelKey) : item.label')
  })
})
