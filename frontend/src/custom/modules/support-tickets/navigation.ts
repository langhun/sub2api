import { h } from 'vue'
import { defineFeatureFlag, makeSidebarFlag } from '@/utils/featureFlags'
import type { CustomNavigationItem } from '../../registry'
const SupportIcon = { render: () => h('svg', { fill: 'none', viewBox: '0 0 24 24', stroke: 'currentColor', 'stroke-width': '1.5' }, [h('path', { d: 'M8.625 10.5h6.75m-6.75 3h4.5M6.75 18.75h-1.5A2.25 2.25 0 013 16.5v-9A2.25 2.25 0 015.25 5.25h13.5A2.25 2.25 0 0121 7.5v9a2.25 2.25 0 01-2.25 2.25h-7.5L7.5 21v-2.25z', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' })]) }
export const supportTicketFeatureFlag = defineFeatureFlag({ key: 'support_ticket_enabled', mode: 'opt-out', label: 'Support Tickets' })
const featureFlag = makeSidebarFlag(supportTicketFeatureFlag)
export const supportTicketNavigation: readonly CustomNavigationItem[] = [
  { path: '/tickets', label: 'Support tickets', labelKey: 'supportTickets.title', icon: SupportIcon, slot: 'after-transfer', featureFlag },
  { path: '/admin/support/tickets', label: 'Support tickets', labelKey: 'admin.supportTickets.title', icon: SupportIcon, section: 'admin', featureFlag },
]
