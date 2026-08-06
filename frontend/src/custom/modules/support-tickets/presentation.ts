import type { SupportTicketCategory, SupportTicketPriority, SupportTicketStatus } from './api'
export type SupportTicketViewer = 'user' | 'admin'
export const ticketStatuses: readonly SupportTicketStatus[] = ['open', 'waiting_user', 'resolved', 'closed']
export const ticketCategories: readonly SupportTicketCategory[] = ['account', 'billing', 'api', 'model', 'feature', 'other']
export const ticketPriorities: readonly SupportTicketPriority[] = ['low', 'normal', 'high', 'urgent']
function needsViewerAttention(value: SupportTicketStatus, viewer: SupportTicketViewer) {
  return (viewer === 'user' && value === 'waiting_user') || (viewer === 'admin' && value === 'open')
}
export function ticketStatusClass(value: SupportTicketStatus, viewer: SupportTicketViewer = 'admin') {
  if (value === 'resolved') return 'badge-success'
  if (value === 'closed') return 'badge-gray'
  return needsViewerAttention(value, viewer) ? 'badge-warning' : 'badge-primary'
}
export function ticketStatusIconClass(value: SupportTicketStatus, viewer: SupportTicketViewer = 'admin') {
  if (value === 'resolved') return 'bg-emerald-50 text-emerald-600 ring-emerald-200 dark:bg-emerald-900/20 dark:text-emerald-300 dark:ring-emerald-900/70'
  if (value === 'closed') return 'bg-gray-100 text-gray-500 ring-gray-200 dark:bg-dark-800 dark:text-dark-400 dark:ring-dark-700'
  return needsViewerAttention(value, viewer)
    ? 'bg-amber-50 text-amber-600 ring-amber-200 dark:bg-amber-900/20 dark:text-amber-300 dark:ring-amber-900/70'
    : 'bg-primary-50 text-primary-600 ring-primary-200 dark:bg-primary-900/20 dark:text-primary-300 dark:ring-primary-900/70'
}
export function ticketStatusBannerClass(value: SupportTicketStatus, viewer: SupportTicketViewer) {
  if (value === 'resolved') return 'border-emerald-200 bg-emerald-50 text-emerald-800 dark:border-emerald-900/70 dark:bg-emerald-900/20 dark:text-emerald-200'
  if (value === 'closed') return 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-300'
  return needsViewerAttention(value, viewer) ? 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/70 dark:bg-amber-900/20 dark:text-amber-200' : 'border-primary-200 bg-primary-50 text-primary-800 dark:border-primary-900/70 dark:bg-primary-900/20 dark:text-primary-200'
}
export function ticketPriorityClass(value: SupportTicketPriority) { return value === 'urgent' ? 'badge-danger' : value === 'high' ? 'badge-warning' : value === 'low' ? 'badge-gray' : 'badge-primary' }
export function ticketStatusLocaleKey(value: SupportTicketStatus, viewer: SupportTicketViewer) {
  return viewer === 'admin' ? `admin.supportTickets.statuses.${value}` : `supportTickets.statuses.${value}`
}
