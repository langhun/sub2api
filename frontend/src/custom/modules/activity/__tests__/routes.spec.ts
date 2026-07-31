import { describe, expect, it } from 'vitest'
import { activityLocaleMessages } from '../locales'
import { activityNavigation } from '../navigation'
import { activityRoutes } from '../routes'

describe('activity module route and navigation contracts', () => {
  it('owns the stable usage, check-in, leaderboard, and red-packet URLs', () => {
    expect(activityRoutes).toEqual(expect.arrayContaining([
      expect.objectContaining({
        path: '/key-usage',
        name: 'KeyUsage',
        meta: expect.objectContaining({ requiresAuth: false, requiresFeature: 'usage_query_enabled' }),
      }),
      expect.objectContaining({
        path: '/usage',
        name: 'Usage',
        meta: expect.objectContaining({ requiresAuth: true, requiresFeature: 'usage_query_enabled' }),
      }),
      expect.objectContaining({
        path: '/checkin',
        name: 'Checkin',
        meta: expect.objectContaining({
          requiresAuth: true,
          requiresAnyFeature: ['checkin_enabled', 'checkin_luck_enabled'],
        }),
      }),
      expect.objectContaining({
        path: '/leaderboard',
        name: 'Leaderboard',
        meta: expect.objectContaining({
          requiresAuth: false,
          requiresFeature: 'leaderboard_enabled',
          requiresAnyFeatureGroups: [
            ['leaderboard_balance_enabled'],
            ['leaderboard_consumption_enabled'],
            ['leaderboard_checkin_enabled'],
            ['leaderboard_transfer_enabled', 'transfer_enabled'],
          ],
        }),
      }),
      expect.objectContaining({
        path: '/transfer/leaderboard',
        name: 'TransferLeaderboard',
        props: { initialTab: 'transfer' },
        meta: expect.objectContaining({
          requiresAuth: true,
          requiresAllFeatures: ['transfer_enabled', 'leaderboard_enabled', 'leaderboard_transfer_enabled'],
        }),
      }),
      expect.objectContaining({
        path: '/redpacket',
        name: 'RedPacket',
        meta: expect.objectContaining({ requiresAuth: true, requiresFeature: 'redpacket_enabled' }),
      }),
    ]))
  })

  it('keeps the activity menu positions in the module contract', () => {
    expect(activityNavigation).toEqual(expect.arrayContaining([
      expect.objectContaining({ path: '/usage', slot: 'after-batch-image' }),
      expect.objectContaining({ path: '/checkin', slot: 'after-affiliate' }),
      expect.objectContaining({ path: '/redpacket', slot: 'after-transfer' }),
      expect.objectContaining({ path: '/leaderboard', slot: 'after-transfer' }),
    ]))
  })

  it('keeps the existing activity message keys available in the module fragment', () => {
    expect(activityLocaleMessages.en).toMatchObject({
      checkin: { title: 'Daily Check-in' },
      leaderboard: { title: 'Leaderboard' },
      redpacket: { title: 'Red Packet Center' },
      activityErrors: { redpacketExpired: 'This packet has expired' },
    })
    expect(activityLocaleMessages.zh).toMatchObject({
      checkin: { title: '每日签到' },
      leaderboard: { title: '排行榜' },
      redpacket: { title: '红包中心' },
      activityErrors: { redpacketExpired: '红包已过期' },
    })
  })
})
