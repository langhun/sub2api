import type { CustomNotificationCenterExtension } from '../../registry'
import SupportTicketNotificationCenterExtension from './components/SupportTicketNotificationCenterExtension.vue'

export const supportTicketNotificationCenterExtensions: readonly CustomNotificationCenterExtension[] = [
  {
    id: 'support-ticket-notifications',
    component: SupportTicketNotificationCenterExtension,
  },
]
