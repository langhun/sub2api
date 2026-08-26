import type { RouteRecordRaw } from 'vue-router'
const userMeta = { requiresAuth: true, requiresAdmin: false, title: 'Support Tickets', titleKey: 'supportTickets.title', requiresFeature: 'support_ticket_enabled' }
const adminMeta = { requiresAuth: true, requiresAdmin: true, title: 'Support Tickets', titleKey: 'admin.supportTickets.title', requiresFeature: 'support_ticket_enabled' }
export const supportTicketRoutes: readonly RouteRecordRaw[] = [
  { path: '/tickets', name: 'SupportTickets', component: () => import('./SupportTicketsView.vue'), meta: userMeta },
  { path: '/tickets/:id', name: 'SupportTicketDetail', component: () => import('./SupportTicketsView.vue'), meta: userMeta },
  { path: '/admin/support/tickets', name: 'AdminSupportTickets', component: () => import('./admin/AdminSupportTicketsView.vue'), meta: adminMeta },
  { path: '/admin/support/tickets/:id', name: 'AdminSupportTicketDetail', component: () => import('./admin/AdminSupportTicketsView.vue'), meta: adminMeta },
]
