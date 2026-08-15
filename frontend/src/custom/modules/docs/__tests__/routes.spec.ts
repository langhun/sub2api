import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

import { customNavigation, customRoutes } from '@/custom/registry'
import router from '@/router'
import { docsNavigation } from '../navigation'
import { docsRoutes } from '../routes'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../../../../router/index.ts'), 'utf8')

describe('documentation overlay integration', () => {
  it('registers authenticated topic routes and user navigation', () => {
    expect(docsRoutes.map((route) => route.path)).toEqual(['/docs', '/docs/:topic'])
    expect(docsRoutes.every((route) => route.meta?.requiresAuth === true)).toBe(true)
    expect(docsRoutes.every((route) => route.meta?.hideHeaderTitle === true)).toBe(true)
    expect(customRoutes).toEqual(expect.arrayContaining(docsRoutes))
    expect(customNavigation).toEqual(expect.arrayContaining(docsNavigation))
    expect(docsNavigation[0]).toMatchObject({ path: '/docs', labelKey: 'nav.helpDocs', slot: 'after-affiliate' })
  })

  it('keeps the existing batch image static route ahead of custom dynamic docs', () => {
    expect(routerSource).toContain("alias: '/docs/batch-image'")
    expect(routerSource.indexOf("alias: '/docs/batch-image'")).toBeLessThan(routerSource.indexOf('...customRoutes'))
    expect(router.resolve('/docs/batch-image').name).toBe('BatchImageGuide')
  })
})
