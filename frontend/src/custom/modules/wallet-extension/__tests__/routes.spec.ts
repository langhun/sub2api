import { describe, expect, it } from 'vitest'
import { walletExtensionLocaleMessages } from '../locales'
import { walletExtensionNavigation } from '../navigation'
import { walletExtensionRoutes } from '../routes'

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

  it('provides a locale fragment only for direct transfers', () => {
    expect(walletExtensionLocaleMessages.zh.transfer.title).toBe('余额转账')
    expect(walletExtensionLocaleMessages.en.transfer).not.toHaveProperty('redpacket')
    expect(walletExtensionLocaleMessages.en.transfer).not.toHaveProperty('leaderboard')
  })
})
