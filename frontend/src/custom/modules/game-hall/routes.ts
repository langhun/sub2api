import type { RouteRecordRaw } from 'vue-router'

export const gameHallRoutes: readonly RouteRecordRaw[] = [
  {
    path: '/game-hall',
    name: 'GameHall',
    component: () => import('./GameHallView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Game Hall',
      titleKey: 'gameHall.title',
      requiresFeature: 'game_hall_enabled',
    },
  },
]
