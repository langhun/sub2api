import type { RouteRecordRaw } from 'vue-router'

export const brandHomeRoutes: readonly RouteRecordRaw[] = [
  {
    path: '/Dino',
    name: 'DaiGua',
    component: () => import('@/custom/modules/brand-home/DaiGuaView.vue'),
    meta: {
      requiresAuth: false,
      title: 'DaiGua',
    },
  },
  {
    path: '/',
    name: 'RootHome',
    component: () => import('@/custom/modules/brand-home/RootHomeView.vue'),
    meta: {
      requiresAuth: false,
      title: 'Home',
    },
  },
]
