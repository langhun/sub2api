import type { CustomHeaderAction } from '../../registry'
import SupportTicketNotificationAction from './components/SupportTicketNotificationAction.vue'

export const supportTicketHeaderActions: readonly CustomHeaderAction[] = [
  {
    id: 'support-ticket-notifications',
    component: SupportTicketNotificationAction,
  },
]
