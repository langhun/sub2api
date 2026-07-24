import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import { customNavigation, customRoutes } from '../registry'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../../router/index.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(directory, '../../components/layout/AppSidebar.vue'), 'utf8')

describe('custom overlay registry', () => {
  it('starts without registered routes or navigation items', () => {
    expect(customRoutes).toEqual([])
    expect(customNavigation).toEqual([])
  })

  it('mounts custom routes before the catch-all route', () => {
    expect(routerSource.indexOf('...customRoutes')).toBeGreaterThan(-1)
    expect(routerSource.indexOf('...customRoutes')).toBeLessThan(routerSource.indexOf("path: '/:pathMatch(.*)*'"))
  })

  it('mounts custom navigation through the sidebar self-navigation builder', () => {
    expect(sidebarSource).toContain('...customNavigation,')
  })
})
