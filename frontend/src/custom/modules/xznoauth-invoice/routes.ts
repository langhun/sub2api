import type { RouteRecordRaw } from 'vue-router'

export const xznoauthInvoiceRoutes: readonly RouteRecordRaw[] = [
  {
    path: '/invoices/apply',
    name: 'InvoiceApply',
    component: () => import('./InvoiceApplyView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Apply for invoice',
      titleKey: 'invoices.applyTitle',
    },
  },
  {
    path: '/invoices',
    name: 'Invoices',
    component: () => import('./InvoicesView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Invoice records',
      titleKey: 'invoices.recordsTitle',
    },
  },
]
