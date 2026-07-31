import type { RouteRecordRaw } from 'vue-router'

export const activityRoutes: readonly RouteRecordRaw[] = [
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
      ],
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
