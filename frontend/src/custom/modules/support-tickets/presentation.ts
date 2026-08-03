import type { SupportTicketCategory, SupportTicketPriority, SupportTicketStatus } from './api'
export type SupportTicketViewer = 'user' | 'admin'
export const ticketStatuses: readonly SupportTicketStatus[] = ['open', 'waiting_user', 'resolved', 'closed']
export const ticketCategories: readonly SupportTicketCategory[] = ['account', 'billing', 'api', 'model', 'feature', 'other']
export const ticketPriorities: readonly SupportTicketPriority[] = ['low', 'normal', 'high', 'urgent']
export function ticketStatusClass(value: SupportTicketStatus) { return value === 'open' ? 'badge-warning' : value === 'waiting_user' ? 'badge-primary' : value === 'resolved' ? 'badge-success' : 'badge-gray' }
export function ticketPriorityClass(value: SupportTicketPriority) { return value === 'urgent' ? 'badge-danger' : value === 'high' ? 'badge-warning' : value === 'low' ? 'badge-gray' : 'badge-primary' }
export function ticketStatusLocaleKey(value: SupportTicketStatus, viewer: SupportTicketViewer) {
  return viewer === 'admin' ? `admin.supportTickets.statuses.${value}` : `supportTickets.statuses.${value}`
}
