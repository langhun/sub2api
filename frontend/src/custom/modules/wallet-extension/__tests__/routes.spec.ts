import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'
import { walletExtensionLocaleMessages } from '../locales'
import { walletExtensionNavigation } from '../navigation'
import { walletExtensionRoutes } from '../routes'

const directory = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(directory, '../../../../router/index.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(directory, '../../../../components/layout/AppSidebar.vue'), 'utf8')

describe('wallet extension overlay contract', () => {
  it('preserves the authenticated feature-gated direct-transfer route', () => {
    expect(walletExtensionRoutes).toContainEqual(expect.objectContaining({
      path: '/transfer',
      name: 'BalanceTransfer',
      meta: expect.objectContaining({
        requiresAuth: true,
        requiresAdmin: false,
        titleKey: 'nav.transfer',
        requiresFeature: 'transfer_enabled',
      }),
    }))
  })

  it('keeps direct-transfer navigation adjacent to affiliate navigation', () => {
    expect(walletExtensionNavigation).toContainEqual(expect.objectContaining({
      path: '/transfer',
      labelKey: 'nav.transfer',
      hideInSimpleMode: true,
      slot: 'after-affiliate',
    }))
  })

  it('registers the existing administrator transfer page through the wallet module', () => {
    expect(walletExtensionRoutes).toContainEqual(expect.objectContaining({
      path: '/admin/transfer',
      name: 'AdminBalanceTransfer',
      meta: expect.objectContaining({
        requiresAuth: true,
        requiresAdmin: true,
        titleKey: 'nav.transferManage',
      }),
    }))
    expect(walletExtensionNavigation).toContainEqual(expect.objectContaining({
      path: '/admin/transfer',
      labelKey: 'nav.transferManage',
      section: 'admin',
      hideInSimpleMode: true,
    }))
  })

  it('keeps the core router and sidebar free of direct administrator transfer entries', () => {
    expect(routerSource).not.toContain("path: '/admin/transfer'")
    expect(sidebarSource).not.toContain("{ path: '/admin/transfer'")
  })

  it('provides a locale fragment only for direct transfers', () => {
    expect(walletExtensionLocaleMessages.zh.transfer.title).toBe('余额转账')
    expect(walletExtensionLocaleMessages.en.transfer).not.toHaveProperty('redpacket')
    expect(walletExtensionLocaleMessages.en.transfer).not.toHaveProperty('leaderboard')
  })
})
