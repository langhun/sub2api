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
]
