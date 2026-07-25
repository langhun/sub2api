import type { RouteRecordRaw } from 'vue-router'

export const walletExtensionRoutes: readonly RouteRecordRaw[] = [
  {
    path: '/transfer',
    name: 'BalanceTransfer',
    component: () => import('./views/DirectTransferView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Balance Transfer',
      titleKey: 'nav.transfer',
      requiresFeature: 'transfer_enabled',
    },
  },
]
