import type { RouteRecordRaw } from 'vue-router'

export const docsRoutes: readonly RouteRecordRaw[] = [
  {
    path: '/docs',
    name: 'CustomDocs',
    component: () => import('./DocsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Help documentation',
      titleKey: 'docs.pageTitle',
    },
  },
  {
    path: '/docs/:topic',
    name: 'CustomDocsTopic',
    component: () => import('./DocsView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Help documentation',
      titleKey: 'docs.pageTitle',
    },
  },
]
