import type { RouteRecordRaw } from 'vue-router'

export const accountDrainRoutes: readonly RouteRecordRaw[] = [
  {
    path: '/admin/account-drain',
    name: 'AccountDrainPlans',
    component: () => import('./views/AccountDrainView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: true,
      title: 'Account Drain Plans',
    },
  },
]
