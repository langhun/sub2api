import type { RouteRecordRaw } from 'vue-router'

export const activityRoutes: readonly RouteRecordRaw[] = [
  {
    path: '/key-usage',
    name: 'KeyUsage',
    component: () => import('@/views/KeyUsageView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Key Usage',
      requiresFeature: 'usage_query_enabled',
    },
  },
  {
    path: '/usage',
    name: 'Usage',
    component: () => import('@/views/user/UsageView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Usage Records',
      titleKey: 'usage.title',
      descriptionKey: 'usage.description',
      requiresFeature: 'usage_query_enabled',
    },
  },
  {
    path: '/checkin',
    name: 'Checkin',
    component: () => import('./views/CheckinView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Check-in',
      titleKey: 'nav.checkin',
      descriptionKey: 'checkin.page.description',
      requiresAnyFeature: ['checkin_enabled', 'checkin_luck_enabled'],
    },
  },
  {
    path: '/leaderboard',
    name: 'Leaderboard',
    component: () => import('./views/LeaderboardView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Leaderboard',
      titleKey: 'leaderboard.title',
      requiresFeature: 'leaderboard_enabled',
      requiresAnyFeatureGroups: [
        ['leaderboard_balance_enabled'],
        ['leaderboard_consumption_enabled'],
        ['leaderboard_checkin_enabled'],
        ['leaderboard_transfer_enabled', 'transfer_enabled'],
      ],
    },
  },
  {
    path: '/transfer/leaderboard',
    name: 'TransferLeaderboard',
    component: () => import('./views/LeaderboardView.vue'),
    props: { initialTab: 'transfer' },
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Transfer Leaderboard',
      titleKey: 'nav.transferLeaderboard',
      requiresAllFeatures: ['transfer_enabled', 'leaderboard_enabled', 'leaderboard_transfer_enabled'],
    },
  },
  {
    path: '/redpacket',
    name: 'RedPacket',
    component: () => import('./views/RedPacketView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Red Packet',
      titleKey: 'nav.redpacket',
      requiresFeature: 'redpacket_enabled',
    },
  },
]
